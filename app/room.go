package app

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/hantabaru1014/instant-playback-sync/dto"
)

type receivedSyncCmd struct {
	syncCmd     *dto.SyncCmd
	received_at time.Time
}

func newReceivedSyncCmd(syncCmd *dto.SyncCmd) *receivedSyncCmd {
	return &receivedSyncCmd{
		syncCmd:     syncCmd,
		received_at: time.Now(),
	}
}

func (rsc *receivedSyncCmd) makeCmdMsgToSend() (*dto.CmdMsg, error) {
	currentTime := rsc.syncCmd.CurrentTime
	if rsc.syncCmd.Event == dto.SYNCCMD_EVENT_PLAY {
		currentTime += (float32(time.Since(rsc.received_at).Seconds()) * rsc.syncCmd.PlaybackRate)
	}
	syncCmd := *rsc.syncCmd
	syncCmd.CurrentTime = currentTime
	jsonBytes, err := json.Marshal(syncCmd)
	if err != nil {
		return nil, err
	}
	return &dto.CmdMsg{
		Command: dto.CMDMSG_CMD_SYNC,
		Payload: json.RawMessage(string(jsonBytes)),
	}, nil
}

type sessionSet struct {
	mu       sync.RWMutex
	sessions map[*Session]struct{}
}

func (ss *sessionSet) add(s *Session) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.sessions[s] = struct{}{}
}

func (ss *sessionSet) del(s *Session) {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	delete(ss.sessions, s)
}

func (ss *sessionSet) clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()

	ss.sessions = make(map[*Session]struct{})
}

func (ss *sessionSet) each(cb func(*Session)) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for s := range ss.sessions {
		cb(s)
	}
}

func (ss *sessionSet) all() []*Session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	sessions := make([]*Session, 0, len(ss.sessions))
	for s := range ss.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

func (ss *sessionSet) getOne() *Session {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	for s := range ss.sessions {
		return s
	}
	return nil
}

func (ss *sessionSet) len() int {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	return len(ss.sessions)
}

type broadcastPacket struct {
	msg     msgPacket
	exclude *Session
}

type Room struct {
	ID string

	server     *Server
	sessions   sessionSet
	broadcast  chan broadcastPacket
	register   chan *Session
	unregister chan *Session
	shutdown   chan []byte

	lastSyncCmdMsg *receivedSyncCmd
}

func NewRoom(id string, s *Server) *Room {
	return &Room{
		ID:         id,
		server:     s,
		sessions:   sessionSet{sessions: make(map[*Session]struct{})},
		broadcast:  make(chan broadcastPacket),
		register:   make(chan *Session),
		unregister: make(chan *Session),
		shutdown:   make(chan []byte),
	}
}

func (r *Room) broadcastOthers(msg []byte, s *Session) {
	r.broadcast <- broadcastPacket{msg: msgPacket{t: websocket.TextMessage, msg: msg}, exclude: s}
}

func (r *Room) handleMessage(msg []byte, s *Session) {
	slog.Debug("handle WS message", "msg", string(msg))
	cmd, err := dto.UnmarshalCmdMsg(msg)
	if err != nil {
		return
	}
	if cmd.Command == dto.CMDMSG_CMD_SYNC {
		syncCmd, err := dto.UnmarshalSyncCmd(cmd.Payload)
		if err != nil {
			return
		}
		r.lastSyncCmdMsg = newReceivedSyncCmd(syncCmd)
	}
	r.broadcastOthers(msg, s)
}

func (r *Room) handleConnect(s *Session) {
	if r.lastSyncCmdMsg != nil {
		cmd, err := r.lastSyncCmdMsg.makeCmdMsgToSend()
		if err != nil {
			return
		}
		cmdJson, err := json.Marshal(cmd)
		if err != nil {
			return
		}
		s.Send(cmdJson)
	} else {
		cmd := &dto.CmdMsg{
			Command: dto.CMDMSG_CMD_REQ_SYNC,
			Payload: nil,
		}
		cmdJson, err := json.Marshal(cmd)
		if err != nil {
			return
		}
		s.Send(cmdJson)
	}
}

func (r *Room) handleDisconnect(s *Session) {
}

// handleSessionError はWSセッションが致命的なエラーを起こして機能しなくなったときに呼ばれる
func (r *Room) handleSessionError(s *Session, err error) {
	slog.Info("Session error", "room_id", r.ID, "session_id", s.ID, "error", err)
}

func (r *Room) sendReqSync() {
	s := r.sessions.getOne()
	if s != nil {
		cmd := &dto.CmdMsg{
			Command: dto.CMDMSG_CMD_REQ_SYNC,
			Payload: nil,
		}
		cmdJson, err := json.Marshal(cmd)
		if err != nil {
			return
		}
		s.Send(cmdJson)
	}
}

func (r *Room) run() {
	ticker := time.NewTicker(r.server.Config.SyncInterval)
	defer ticker.Stop()

loop:
	for {
		select {
		case s := <-r.register:
			r.sessions.add(s)
		case s := <-r.unregister:
			r.sessions.del(s)
		case m := <-r.broadcast:
			r.sessions.each(func(s *Session) {
				if s != m.exclude {
					s.sendMsg(m.msg)
				}
			})
		case m := <-r.shutdown:
			r.sessions.each(func(s *Session) {
				s.CloseWithMsg(m)
			})
			r.sessions.clear()

			break loop
		case <-ticker.C:
			r.sendReqSync()
		}
	}
}

func (r *Room) handleRequest(conn *websocket.Conn, id string) error {
	s := &Session{
		ID:         id,
		room:       r,
		conn:       conn,
		output:     make(chan msgPacket, r.server.Config.WSMessageSendBufferSize),
		outputDone: make(chan struct{}),
		opened:     true,
		rwmutex:    &sync.RWMutex{},
	}
	r.register <- s
	r.handleConnect(s)

	go s.writePump()
	s.readPump()

	r.unregister <- s
	s.close()
	r.handleDisconnect(s)

	return nil
}

func (r *Room) all() []*Session {
	return r.sessions.all()
}

func (r *Room) Shutdown(msg []byte) {
	r.shutdown <- msg
}

func (r *Room) Len() int {
	return r.sessions.len()
}

func (r *Room) ToDTO() *dto.Room {
	var url *string
	if r.lastSyncCmdMsg != nil {
		url = &r.lastSyncCmdMsg.syncCmd.PageUrl
	}
	d := dto.Room{
		ID:       r.ID,
		VideoUrl: url,
	}
	return &d
}

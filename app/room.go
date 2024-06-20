package app

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/hantabaru1014/instant-playback-sync/dto"
	"github.com/olahol/melody"
)

type roomMap struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func newRoomMap() *roomMap {
	return &roomMap{
		rooms: make(map[string]*Room),
	}
}

func (rm *roomMap) del(id string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	delete(rm.rooms, id)
}

func (rm *roomMap) getOrAdd(id string, m *melody.Melody) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	if r, ok := rm.rooms[id]; ok {
		return r
	}

	r := &Room{
		ID: id,
		m:  m,
	}
	rm.rooms[id] = r
	return r
}

func (rm *roomMap) get(id string) *Room {
	if r, ok := rm.rooms[id]; ok {
		return r
	} else {
		return nil
	}
}

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

type Room struct {
	ID string

	m              *melody.Melody
	lastSyncCmdMsg *receivedSyncCmd
}

func (r *Room) broadcastOthers(msg []byte, s *melody.Session) error {
	return r.m.BroadcastFilter(msg, func(ss *melody.Session) bool {
		room, exists := ss.Get(ROOM_KEY)
		return exists && room == r && ss != s
	})
}

func (r *Room) isEmptyOrErr(exclude *melody.Session) bool {
	sess, err := r.m.Sessions()
	if err != nil {
		return true
	}
	for _, s := range sess {
		if s == exclude {
			continue
		}
		if room, exists := s.Get(ROOM_KEY); exists && room == r {
			return false
		}
	}
	return true
}

func (r *Room) handleMessage(cmd *dto.CmdMsg, rawMsg []byte, s *melody.Session) {
	if cmd.Command == dto.CMDMSG_CMD_SYNC {
		syncCmd, err := dto.UnmarshalSyncCmd(cmd.Payload)
		if err != nil {
			return
		}
		r.lastSyncCmdMsg = newReceivedSyncCmd(syncCmd)
	}
	r.broadcastOthers(rawMsg, s)
}

func (r *Room) handleConnect(s *melody.Session) {
	if r.lastSyncCmdMsg != nil {
		cmd, err := r.lastSyncCmdMsg.makeCmdMsgToSend()
		if err != nil {
			return
		}
		cmdJson, err := json.Marshal(cmd)
		if err != nil {
			return
		}
		s.Write(cmdJson)
	}
}

func (r *Room) ToDTO() *dto.RoomDTO {
	var url *string
	if r.lastSyncCmdMsg != nil {
		url = &r.lastSyncCmdMsg.syncCmd.PageUrl
	}
	d := dto.RoomDTO{
		ID:       r.ID,
		VideoUrl: url,
	}
	return &d
}

package app

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type msgPacket struct {
	t   int // message type
	msg []byte
}

type Session struct {
	ID string

	room       *Room
	conn       *websocket.Conn
	output     chan msgPacket
	outputDone chan struct{}
	opened     bool
	rwmutex    *sync.RWMutex
}

func (s *Session) sendMsg(msg msgPacket) {
	select {
	case s.output <- msg:
	default:
		s.room.handleSessionError(s, ErrSessionSendBufferFull)
	}
}

func (s *Session) makePongDeadline() time.Time {
	return time.Now().Add(s.room.server.Config.WSPongWait)
}

func (s *Session) writeRaw(msg msgPacket) error {
	if s.closed() {
		return ErrSessionClosed
	}
	s.conn.SetWriteDeadline(time.Now().Add(s.room.server.Config.WSSendWait))
	err := s.conn.WriteMessage(msg.t, msg.msg)
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) ping() {
	s.writeRaw(msgPacket{t: websocket.PingMessage, msg: []byte{}})
}

func (s *Session) close() {
	s.rwmutex.Lock()
	open := s.opened
	s.opened = false
	s.rwmutex.Unlock()

	if open {
		s.conn.Close()
		close(s.outputDone)
	}
}

func (s *Session) closed() bool {
	s.rwmutex.RLock()
	defer s.rwmutex.RUnlock()

	return !s.opened
}

func (s *Session) readPump() {
	s.conn.SetReadLimit(s.room.server.Config.WSMaxMessageSize)
	s.conn.SetReadDeadline(s.makePongDeadline())
	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(s.makePongDeadline())
		return nil
	})

	for {
		t, msg, err := s.conn.ReadMessage()
		if err != nil {
			s.room.handleSessionError(s, err)
			break
		}
		if t == websocket.TextMessage {
			s.room.handleMessage(msg, s)
		}
	}
}

func (s *Session) writePump() {
	ticker := time.NewTicker(s.room.server.Config.WSPingPeriod)
	defer ticker.Stop()

loop:
	for {
		select {
		case m := <-s.output:
			err := s.writeRaw(m)
			if err != nil {
				s.room.handleSessionError(s, err)
				break loop
			}
			if m.t == websocket.CloseMessage {
				break loop
			}
		case <-ticker.C:
			s.ping()
		case _, ok := <-s.outputDone:
			if !ok {
				break loop
			}
		}
	}

	s.close()
}

func (s *Session) Send(msg []byte) error {
	if s.closed() {
		return ErrSessionClosed
	}
	s.sendMsg(msgPacket{t: websocket.TextMessage, msg: msg})
	return nil
}

func (s *Session) CloseWithMsg(msg []byte) {
	if s.closed() {
		return
	}
	s.sendMsg(msgPacket{t: websocket.CloseMessage, msg: msg})
}

func (s *Session) IsClosed() bool {
	return s.closed()
}

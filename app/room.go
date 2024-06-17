package app

import (
	"sync"

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

type Room struct {
	ID string

	m              *melody.Melody
	lastSyncCmdMsg *[]byte
	lastVideoUrl   *string
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
	if cmd.Command == "sync" {
		r.lastSyncCmdMsg = &rawMsg

		syncCmd, err := dto.UnmarshalSyncCmd(cmd.Payload)
		if err != nil {
			return
		}
		v := syncCmd.PageUrl
		r.lastVideoUrl = &v
	}
	r.broadcastOthers(rawMsg, s)
}

func (r *Room) handleConnect(s *melody.Session) {
	if r.lastSyncCmdMsg != nil {
		s.Write(*r.lastSyncCmdMsg)
	}
}

func (r *Room) ToDTO() *dto.RoomDTO {
	d := dto.RoomDTO{
		ID:       r.ID,
		VideoUrl: r.lastVideoUrl,
	}
	return &d
}

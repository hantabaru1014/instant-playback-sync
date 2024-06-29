package app

import (
	"sync"
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

func (rm *roomMap) get(id string) *Room {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if r, ok := rm.rooms[id]; ok {
		return r
	} else {
		return nil
	}
}

func (rm *roomMap) add(r *Room) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.rooms[r.ID] = r
}

func (rm *roomMap) each(cb func(*Room)) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	for _, r := range rm.rooms {
		cb(r)
	}
}

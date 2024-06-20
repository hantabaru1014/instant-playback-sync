package dto

import "encoding/json"

type SyncCmdEvent string

const (
	SYNCCMD_EVENT_PLAY = SyncCmdEvent("play")
)

type SyncCmd struct {
	PageUrl      string       `json:"pageUrl"`
	Event        SyncCmdEvent `json:"event"`
	CurrentTime  float32      `json:"currentTime"`
	PlaybackRate float32      `json:"playbackRate"`
}

func UnmarshalSyncCmd(data []byte) (*SyncCmd, error) {
	var s SyncCmd
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	// TODO: validate s.Event
	return &s, nil
}

package dto

import "encoding/json"

type CmdMsg struct {
	Command string          `json:"cmd"`
	Payload json.RawMessage `json:"p"`
}

func UnmarshalCmdMsg(data []byte) (*CmdMsg, error) {
	var c CmdMsg
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	// TODO: validate c.Command
	return &c, nil
}

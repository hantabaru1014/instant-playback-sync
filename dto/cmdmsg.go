package dto

import "encoding/json"

type CmdMsgCommand string

const (
	CMDMSG_CMD_SYNC = CmdMsgCommand("sync")
)

type CmdMsg struct {
	Command CmdMsgCommand   `json:"cmd"`
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

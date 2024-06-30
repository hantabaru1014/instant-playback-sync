package app

import "time"

type Config struct {
	SyncInterval            time.Duration // イベントがなくても同期を行う間隔
	WSSendWait              time.Duration // WSメッセージ送信のタイムアウト
	WSPongWait              time.Duration // WS 応答のタイムアウト
	WSPingPeriod            time.Duration // WS Ping送信の間隔
	WSMaxMessageSize        int64         // WSメッセージの最大サイズ(bytes)
	WSMessageSendBufferSize int           // WSメッセージ送信バッファサイズ(あふれるとメッセージをドロップする)
}

func NewConfig() *Config {
	return &Config{
		SyncInterval:            60 * time.Second,
		WSSendWait:              10 * time.Second,
		WSPongWait:              40 * time.Second,
		WSPingPeriod:            30 * time.Second,
		WSMaxMessageSize:        1024,
		WSMessageSendBufferSize: 16,
	}
}

package app

import "errors"

var (
	ErrSessionClosed         = errors.New("session is already closed")
	ErrSessionSendBufferFull = errors.New("session send buffer is full")
)

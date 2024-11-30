package storage

import (
	"errors"

	"api.com/quick/pkg/messages"
)

var (
	ErrNotFound = errors.New("message not found")
)

type Storage interface {
	Store(messages.Message) error
	Load(messages.MsgID) (messages.Message, error)
	All() ([]messages.Message, error)
}

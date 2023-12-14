package socket

import (
	"github.com/HuolalaTech/page-spy-api/api/room"
)

func Unwrap(err error) error {
	u, ok := err.(interface {
		Unwrap() error
	})
	if !ok {
		return err
	}
	return Unwrap(u.Unwrap())
}

func NewErrorMessage(err error) *room.Message {
	te := Unwrap(err)
	re, ok := te.(*room.Error)
	if ok {
		return &room.Message{
			Type:    room.ErrorType,
			Content: &room.ErrorMessageContent{Message: re.Message, Code: string(re.Code)},
		}
	}

	return &room.Message{
		Type:    room.ErrorType,
		Content: &room.ErrorMessageContent{Message: te.Error(), Code: room.UnknownError},
	}
}

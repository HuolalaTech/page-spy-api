package room

import "fmt"

type Error struct {
	Code    ErrorCode
	Message string
}
type BasicRpcResponseInterface interface {
	GetError() *Error
}

type BasicRpcResponse struct {
	Error *Error
}

func (b *BasicRpcResponse) GetError() *Error {
	return b.Error
}

func (b *BasicRpcResponse) SetError(err error) error {
	if err == nil {
		return nil
	}

	e, ok := err.(*Error)
	if ok {
		b.Error = e
		return nil
	}
	b.Error = NewError(err.Error())
	return nil
}

type ErrorCode string

const (
	UnknownError        = "UnknownError"
	RoomNotFoundError   = "RoomNotFoundError"
	RoomCloseError      = "RoomCloseError"
	NetWorkTimeoutError = "NetWorkTimeoutError"
	MessageContentError = "MessageContentError"
	ServeError          = "ServeError"
)

func (e Error) Error() string {
	return e.Message
}

func NewError(msg string) *Error {
	return &Error{
		Code:    UnknownError,
		Message: msg,
	}
}

func NewErrorWithCode(msg string, code ErrorCode) *Error {
	return &Error{
		Code:    code,
		Message: msg,
	}
}

func NewRoomNotFoundError(msg string, a ...any) *Error {
	return NewErrorWithCode(fmt.Sprintf(msg, a...), RoomNotFoundError)
}

func NewMessageContentError(msg string, a ...any) *Error {
	return NewErrorWithCode(fmt.Sprintf(msg, a...), MessageContentError)
}

func NewRoomCloseError(msg string, a ...any) *Error {
	return NewErrorWithCode(fmt.Sprintf(msg, a...), RoomCloseError)
}

func NewNetWorkTimeoutError(msg string, a ...any) *Error {
	return NewErrorWithCode(fmt.Sprintf(msg, a...), NetWorkTimeoutError)
}

func NewServeError(msg string, a ...any) *Error {
	return NewErrorWithCode(fmt.Sprintf(msg, a...), ServeError)
}

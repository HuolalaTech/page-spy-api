package room

import "fmt"

type BasicRpcResponseInterface interface {
	GetError() error
}

type BasicRpcResponse struct {
	Error *Error `json:"error"`
}

func (b *BasicRpcResponse) GetError() error {
	if b.Error == nil {
		return nil
	}

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
	ClientError         = "ClientError"
)

type Error struct {
	Code    ErrorCode
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("Code: %s Message: %s", e.Code, e.Message)
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

func NewClientError(msg string, a ...any) *Error {
	return NewErrorWithCode(fmt.Sprintf(msg, a...), ClientError)
}

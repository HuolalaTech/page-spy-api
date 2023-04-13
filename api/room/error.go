package room

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
	UnknownError      = "UnknownError"
	RoomNotFoundError = "RoomNotFoundError"
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

func NewRoomNotFoundError(msg string) *Error {
	return NewErrorWithCode(msg, RoomNotFoundError)
}

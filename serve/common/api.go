package common

import (
	"github.com/HuolalaTech/page-spy-api/api/room"
)

type Response struct {
	Code    string      `json:"code"`
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
}

func NewErrorResponse(err error) *Response {
	re, ok := err.(*room.Error)
	if ok {
		return &Response{
			Code:    string(re.Code),
			Success: false,
			Message: err.Error(),
		}
	}

	return &Response{
		Code:    "error",
		Success: false,
		Message: err.Error(),
	}
}

func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    "success",
		Data:    data,
		Success: true,
	}
}

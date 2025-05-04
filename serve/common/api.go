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

// NewErrorResponseWithCode 创建包含自定义错误码的错误响应
func NewErrorResponseWithCode(message string, code string) *Response {
	resp := &Response{
		Code:    code,
		Success: false,
		Message: message,
	}

	// 特殊处理：密码未设置的情况
	if code == "PASSWORD_REQUIRED" {
		resp.Data = map[string]interface{}{
			"needPasswordSetup": true,
		}
	}

	return resp
}

func NewSuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    "success",
		Data:    data,
		Success: true,
	}
}

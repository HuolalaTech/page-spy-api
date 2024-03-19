package data

import (
	"gorm.io/gorm"
)

type Status string

const (
	Created Status = "已创建"
	Error   Status = "错误"
	Saved   Status = "已保存"
	Unknown Status = "未知"
)

type LogData struct {
	gorm.Model
	Status Status
	Size   int64
	Name   string
}

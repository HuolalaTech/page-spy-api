package data

import (
	"gorm.io/gorm"
)

type Status string

const (
	Error   Status = "错误"
	Created Status = "已创建"
	Saved   Status = "已保存"
	Unknown Status = "未知"
)

type LogData struct {
	gorm.Model
	Status Status
	Size   int64
	FileId string
	Name   string
}

package data

import (
	"time"

	"gorm.io/gorm"
)

type Status string

const (
	Error   Status = "错误"
	Created Status = "已创建"
	Saved   Status = "已保存"
	Unknown Status = "未知"
)

type Page[T any] struct {
	Total int64 `json:"total"`
	Data  []T   `json:"data"`
}

type Model struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

type LogData struct {
	Model
	Status Status `json:"status"`
	Size   int64  `json:"size"`
	FileId string `gorm:"index:unique" json:"fileId"`
	Name   string `json:"name"`
}

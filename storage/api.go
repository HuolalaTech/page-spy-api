package storage

import (
	"fmt"
	"io"
	"os"

	"github.com/HuolalaTech/page-spy-api/data"
)

type LogFile struct {
	Name      string        `json:"name"`
	FileId    string        `json:"fileId"`
	Size      int64         `json:"size"`
	Tags      []*data.Tag   `json:"tags"`
	File      []byte        `json:"-"`
	FileSteam io.ReadCloser `json:"-"`
}

type StorageApi interface {
	SaveLog(log *LogFile) error
	GetLog(fileId string) (*LogFile, error)
	RemoveLog(fileId string) error
}

func NewStorage() (StorageApi, error) {
	if err := os.MkdirAll(logDirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("init log file dir error: %w", err)
	}
	return &FileApi{}, nil
}

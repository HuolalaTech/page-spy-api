package storage

import (
	"fmt"
	"io"
	"os"

	"github.com/HuolalaTech/page-spy-api/config"
)

type Tag struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type LogFile struct {
	Name       string        `json:"name"`
	FileId     string        `json:"fileId"`
	Size       int64         `json:"size"`
	Tags       []*Tag        `json:"tags"`
	UpdateFile []byte        `json:"-"`
	FileSteam  io.ReadCloser `json:"-"`
}

type LogGroupFile struct {
	LogFile
	GroupId   string `json:"groupId"`
	GroupName string `json:"groupName"`
}

type StorageApi interface {
	SaveLog(log *LogFile) error
	GetLog(fileId string) (*LogFile, error)
	RemoveLog(fileId string) error
}

func NewStorage(config *config.Config) (StorageApi, error) {
	if config.IsRemoteStorage() {
		return NewS3Api(config.StorageConfig)
	}

	return NewFileApi()
}

func NewS3Api(config *config.StorageConfig) (StorageApi, error) {
	return &RemoteApi{config: config}, nil
}

func NewFileApi() (StorageApi, error) {
	if err := os.MkdirAll(logDirPath, os.ModePerm); err != nil {
		return nil, fmt.Errorf("init log file dir error: %w", err)
	}

	return &FileApi{}, nil
}

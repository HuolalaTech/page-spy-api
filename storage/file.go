package storage

import (
	"fmt"
	"os"
)

const logDirPath = "./log"

type FileApi struct {
}

func (f *FileApi) SaveLog(log *LogFile) error {
	return nil
}

func (f *FileApi) getLog(name string) (*LogFile, error) {
	return nil, nil
}

func (f *FileApi) RemoveLog(name string) error {
	return nil
}

var fileApi StorageApi = &FileApi{}

func init() {
	if err := os.MkdirAll(logDirPath, os.ModePerm); err != nil {
		panic(fmt.Errorf("init log file dir error: %w", err))
	}
}

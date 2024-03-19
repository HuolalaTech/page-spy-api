package storage

import "io"

type LogFile struct {
	Name string
	File io.Reader
}

type StorageApi interface {
	SaveLog(log *LogFile) error
	getLog(name string) (*LogFile, error)
	RemoveLog(name string) error
}

func GetApi() StorageApi {
	return fileApi
}

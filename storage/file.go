package storage

import (
	"fmt"
	"io"
	"os"
)

const logDirPath = "./log"

type FileApi struct {
}

func (f *FileApi) SaveLog(log *LogFile) error {
	if log.FileId == "" {
		return fmt.Errorf("create log file error: fileId is empty")
	}

	filePath := fmt.Sprintf("%s/%s", logDirPath, log.FileId)
	findFile, err := os.Stat(filePath)
	if err == nil && findFile != nil {
		return nil
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create log file error: %w", err)
	}

	defer dst.Close()
	if _, err = dst.Write(log.UpdateFile); err != nil {
		return fmt.Errorf("create log file error: %w", err)
	}

	return nil
}

func (f *FileApi) ExistLog(fileId string) (bool, error) {
	if fileId == "" {
		return false, fmt.Errorf("get log file error: fileId is empty")
	}

	logFilePath := fmt.Sprintf("%s/%s", logDirPath, fileId)

	return f.Exist(logFilePath)
}

func (f *FileApi) Exist(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("get path error: path is empty")
	}

	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("get file size error: %w", err)
	}

	return true, nil
}

func (f *FileApi) GetLog(fileId string) (*LogFile, error) {
	if fileId == "" {
		return nil, fmt.Errorf("get log file error: fileId is empty")
	}

	logFilePath := fmt.Sprintf("%s/%s", logDirPath, fileId)

	fileSteam, fileSize, err := f.Get(logFilePath)
	if err != nil {
		return nil, err
	}

	return &LogFile{
		FileId:    fileId,
		Size:      fileSize,
		FileSteam: fileSteam,
	}, nil
}

func (f *FileApi) Get(path string) (io.ReadCloser, int64, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, 0, fmt.Errorf("get file size error: %w", err)
	}

	fileSteam, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("open log file error: %w", err)
	}

	return fileSteam, fileInfo.Size(), nil
}

func (f *FileApi) RemoveLog(fileId string) error {
	if fileId == "" {
		return fmt.Errorf("remove log file error: fileId is empty")
	}

	filePath := fmt.Sprintf("%s/%s", logDirPath, fileId)
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil
	}

	err = os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("remove log file error: %w", err)
	}

	return nil
}

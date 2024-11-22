package storage

import (
	"fmt"
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

	_, err := os.Stat(logFilePath)
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

	fileInfo, err := os.Stat(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("get file size error: %w", err)
	}

	fileSteam, err := os.Open(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("open log file error: %w", err)
	}

	return &LogFile{
		FileId:    fileId,
		Size:      fileInfo.Size(),
		FileSteam: fileSteam,
	}, nil
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

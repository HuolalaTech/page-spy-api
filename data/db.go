package data

import (
	"fmt"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Data struct {
	db *gorm.DB
}

func NewData() (DataApi, error) {
	db, err := gorm.Open(sqlite.Open("data.db"), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect database")
	}

	if err := db.AutoMigrate(&LogData{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database")
	}

	return &Data{db: db}, nil
}

func (d *Data) CreateLog(log *LogData) error {
	return nil
}

func (d *Data) FindLogs(size int64, page int64) (*Page[*LogData], error) {
	return nil, nil
}

func (d *Data) UpdateLogStatus(fileId string, status Status) error {
	return nil
}

func (d *Data) DeleteLogByName(name string) error {
	return nil
}

func (d *Data) FindLogByName(name string) (*LogData, error) {
	return nil, nil
}

func (d *Data) FindTimeoutLogs(before time.Time) ([]*LogData, error) {
	return nil, nil
}

func (d *Data) FindOldestLogs() ([]*LogData, error) {
	return nil, nil
}

func (d *Data) CountLogsSize() (int64, error) {
	return 0, nil
}

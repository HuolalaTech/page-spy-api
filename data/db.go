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
	result := d.db.Create(log)
	return result.Error
}

func (d *Data) FindLogs(size int, page int) (*Page[*LogData], error) {
	offset := (page - 1) * size
	var logs []*LogData
	result := d.db.Offset(offset).Limit(size).Order("createdAt desc").Find(&logs)
	if result.Error != nil {
		return nil, result.Error
	}

	var total int64
	result = d.db.Model(&LogData{}).Count(&total)
	if result.Error != nil {
		return nil, result.Error
	}

	return &Page[*LogData]{
		Data:  logs,
		Total: total,
	}, nil
}

func (d *Data) UpdateLogStatus(fileId string, status Status) error {
	result := d.db.Where("fileId = ?", fileId).Update("status", status)
	return result.Error
}

func (d *Data) DeleteLogByFileId(fileId string) error {
	result := d.db.Where("fileId = ?", fileId).Delete(&LogData{})
	return result.Error
}

func (d *Data) FindLogByFileId(FileId string) (*LogData, error) {
	log := &LogData{}
	result := d.db.Where("fileId = ?", FileId).First(log)
	return log, result.Error
}

func (d *Data) FindTimeoutLogs(before time.Time, size int) ([]*LogData, error) {
	var logs []*LogData
	result := d.db.Where("createdAt < ?", before).Limit(size).Order("createdAt desc").Find(&logs)
	return logs, result.Error
}

func (d *Data) FindOldestLogs(size int) ([]*LogData, error) {
	var logs []*LogData
	result := d.db.Limit(size).Order("createdAt desc").Find(&logs)
	return logs, result.Error
}

type Sum struct {
	Total int64
}

func (d *Data) CountLogsSize() (int64, error) {
	sum := &Sum{}
	result := d.db.Model(&LogData{}).Select("sum(size) as total").Scan(sum)
	if result.Error != nil {
		return 0, result.Error
	}

	return sum.Total, nil
}

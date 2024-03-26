package data

import (
	"errors"
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

	if err := db.AutoMigrate(&LogData{}, &Tag{}); err != nil {
		return nil, fmt.Errorf("failed to auto migrate database")
	}

	return &Data{db: db}, nil
}

func (d *Data) CreateLog(log *LogData) error {
	findFile, err := d.FindLogByFileId(log.FileId)

	if err != nil {
		return err
	}

	if findFile != nil {
		return nil
	}

	result := d.db.Create(log)
	return result.Error
}

type PageQuery struct {
	Size int
	Page int
}

func (p *PageQuery) GetOffset() int {
	return (p.Page - 1) * p.Size
}

type FileListQuery struct {
	PageQuery
	From    *int64
	To      *int64
	Tags    []*Tag
	Keyword string
}

func (f *FileListQuery) GetFrom() *time.Time {
	if f.From == nil {
		return nil
	}

	from := time.Unix(*f.From, 0)

	return &from
}

func (f *FileListQuery) GetTo() *time.Time {
	if f.To == nil {
		return nil
	}

	to := time.Unix(*f.To, 0)

	return &to
}

func (query *FileListQuery) getDB(db *gorm.DB) *gorm.DB {
	offset := query.GetOffset()
	q := db
	if (query.Tags != nil && len(query.Tags) > 0) || query.Keyword != "" {
		q = q.Joins("join log_tags on log_tags.log_data_id = log_data.id").Joins("join tags on tags.id = log_tags.tag_id")
		for _, tag := range query.Tags {
			q = q.Where("tags.key = ? and tags.value = ?", tag.Key, tag.Value)
		}

		keyword := "%" + query.Keyword + "%"
		q = q.Where("tags.value like ?", keyword)
	}

	q = q.Preload("Tags").Offset(offset).Limit(query.Size).Order("log_data.created_at desc")
	from := query.GetFrom()
	if from != nil {
		q = q.Where("log_data.created_at > ?", from)
	}

	to := query.GetTo()
	if to != nil {
		q = q.Where("log_data.created_at < ?", to)
	}

	return q.Offset(offset).Limit(query.Size).Order("log_data.created_at desc")
}

func (d *Data) FindLogs(query *FileListQuery) (*Page[*LogData], error) {
	if query.Size <= 0 {
		return nil, fmt.Errorf("size should be greater than 0")
	}

	if query.Page <= 0 {
		return nil, fmt.Errorf("page should be greater than 0")
	}

	var logs []*LogData
	result := query.getDB(d.db).Find(&logs)
	if result.Error != nil {
		return nil, result.Error
	}

	var total int64
	result = query.getDB(d.db).Model(&LogData{}).Count(&total)
	if result.Error != nil {
		return nil, result.Error
	}

	return &Page[*LogData]{
		Data:  logs,
		Total: total,
	}, nil
}

func (d *Data) UpdateLogStatus(fileId string, status Status) error {
	result := d.db.Model(&LogData{}).Where("file_id = ?", fileId).Update("status", status)
	return result.Error
}

func (d *Data) DeleteLogByFileId(fileId string) error {
	result := d.db.Where("file_id = ?", fileId).Delete(&LogData{})
	return result.Error
}

func (d *Data) FindLogByFileId(FileId string) (*LogData, error) {
	log := &LogData{}
	result := d.db.Where("file_id = ?", FileId).Where("status = ?", Saved).First(log)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return log, result.Error
}

func (d *Data) FindTimeoutLogs(before time.Time, size int) ([]*LogData, error) {
	var logs []*LogData
	result := d.db.Where("created_at < ?", before).Limit(size).Order("created_at desc").Find(&logs)
	return logs, result.Error
}

func (d *Data) FindOldestLogs(size int) ([]*LogData, error) {
	var logs []*LogData
	result := d.db.Limit(size).Order("created_at asc").Find(&logs)
	return logs, result.Error
}

func (d *Data) FindShouldDeleteLogs(size int) ([]*LogData, error) {
	var logs []*LogData
	status := []Status{
		Error,
		Created,
		Unknown,
	}

	result := d.db.Limit(size).
		Where("created_at < ?", time.Now().Add(-time.Hour*1)).
		Where("status in ?", status).Find(&logs)
	return logs, result.Error
}

type Sum struct {
	Total int64
}

func (d *Data) CountLogsSize() (int64, error) {
	sum := &Sum{}
	result := d.db.Model(&LogData{}).
		Where("status = ?", Saved).
		Select("sum(size) as total").Scan(sum)
	if result.Error != nil {
		return 0, result.Error
	}

	return sum.Total, nil
}

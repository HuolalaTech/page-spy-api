package data

type PageLogData struct {
	Model
	PageId string     `gorm:"index:unique" json:"pageId"`
	Files  []*LogData `json:"files"`
	Size   int64      `json:"size"`
	FileId string     `gorm:"index:unique" json:"fileId"`
	Tags   []*Tag     `gorm:"many2many:page_log_tags;" json:"tags"`
}

func (l *PageLogData) GetOrderValue() float64 {
	return float64(l.CreatedAt.Unix())
}

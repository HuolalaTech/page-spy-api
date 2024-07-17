package data

type Status string

const (
	Error   Status = "Error"
	Created Status = "Created"
	Saved   Status = "Saved"
	Unknown Status = "Unknown"
)

type LogData struct {
	Model
	Status Status `json:"status"`
	Size   int64  `json:"size"`
	FileId string `gorm:"index:unique" json:"fileId"`
	Tags   []*Tag `gorm:"many2many:log_tags;" json:"tags"`
	Name   string `json:"name"`
}

func (l *LogData) GetOrderValue() float64 {
	return float64(l.CreatedAt.Unix())
}

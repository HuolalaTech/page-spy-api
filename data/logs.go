package data

import (
	"fmt"
	"sort"
	"time"

	"github.com/HuolalaTech/page-spy-api/rpc"
	"gorm.io/gorm"
)

type Status string

const (
	Error   Status = "Error"
	Created Status = "Created"
	Saved   Status = "Saved"
	Unknown Status = "Unknown"
)

type Page[T OrderData] struct {
	Total int64 `json:"total"`
	Data  []T   `json:"data"`
}

type OrderData interface {
	GetOrderValue() float64
	GetUniqKey() string
}

func (p *Page[T]) Merge(result rpc.MergeResult) error {
	page, ok := result.(*Page[T])
	if !ok {
		return fmt.Errorf("type error")
	}

	p.Data = append(p.Data, page.Data...)
	p.Total += page.Total
	return nil
}

func (p *Page[T]) Desc() {
	sort.Slice(p.Data, func(i, j int) bool {
		return p.Data[i].GetOrderValue() > p.Data[j].GetOrderValue()
	})
}

func (p *Page[T]) UniqData() {
	seen := make(map[string]struct{})
	clonedData := make([]T, 0, len(p.Data))

	for _, item := range p.Data {
		itemKey := item.GetUniqKey()
		if _, exists := seen[itemKey]; !exists {
			seen[itemKey] = struct{}{}
			clonedData = append(clonedData, item)
		}
	}
	p.Data = clonedData
}

func (p *Page[T]) New() rpc.MergeResult {
	return &Page[T]{}
}

type Model struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (m *Model) GetOrderValue() float64 {
	return float64(m.CreatedAt.Unix())
}

func (m *Model) GetUniqKey() string {
	return fmt.Sprintf("%d_%d", m.ID, m.CreatedAt.Unix())
}

type LogData struct {
	Model
	Status     Status `json:"status"`
	Size       int64  `json:"size"`
	FileId     string `gorm:"index:unique" json:"fileId"`
	LogGroupID *uint  `json:"-" gorm:"index;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Tags       []*Tag `gorm:"many2many:log_tags;" json:"tags"`
	Name       string `json:"name"`
}

type LogGroup struct {
	Model
	GroupId string     `json:"groupId"`
	Tags    []*Tag     `gorm:"many2many:log_group_tags;" json:"tags"`
	Size    int64      `json:"size"`
	Logs    []*LogData `gorm:"foreignKey:LogGroupID" json:"logs"`
	Name    string     `json:"name"`
}

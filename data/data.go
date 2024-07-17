package data

import (
	"fmt"
	"sort"
	"time"

	"github.com/HuolalaTech/page-spy-api/rpc"
	"gorm.io/gorm"
)

type Page[T OrderData] struct {
	Total int64 `json:"total"`
	Data  []T   `json:"data"`
}

type OrderData interface {
	GetOrderValue() float64
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

func (p *Page[T]) New() rpc.MergeResult {
	return &Page[T]{}
}

type Model struct {
	ID        uint           `gorm:"primarykey" json:"-"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

package data

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestMonthGroupExpression(t *testing.T) {
	tests := []struct {
		dialect string
		want    string
	}{
		{dialect: "sqlite", want: "strftime('%Y-%m', log_data.created_at)"},
		{dialect: "mysql", want: "DATE_FORMAT(log_data.created_at, '%Y-%m')"},
	}

	for _, test := range tests {
		t.Run(test.dialect, func(t *testing.T) {
			if got := monthGroupExpression(test.dialect); got != test.want {
				t.Fatalf("month expression = %q, want %q", got, test.want)
			}
		})
	}
}

func TestCountLogsGroupSQLite(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	if err := db.AutoMigrate(&LogGroup{}, &LogData{}, &Tag{}); err != nil {
		t.Fatalf("migrate SQLite: %v", err)
	}

	logData := &LogData{
		Model: Model{
			CreatedAt: time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2024, time.March, 1, 12, 0, 0, 0, time.UTC),
		},
		Status: Saved,
		FileId: "local.file",
		Tags: []*Tag{{
			Key:   "env",
			Value: "prod",
		}},
	}
	if err := db.Create(logData).Error; err != nil {
		t.Fatalf("create log data: %v", err)
	}

	results, err := (&Data{db: db}).CountLogsGroup("env")
	if err != nil {
		t.Fatalf("count logs group: %v", err)
	}
	if len(results) != 1 || results[0].Date != "2024-03" || results[0].Tag != "prod" || results[0].Total != 1 {
		t.Fatalf("unexpected count result: %#v", results)
	}
}

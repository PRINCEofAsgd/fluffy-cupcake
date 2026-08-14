package repository

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// TestAddClicksUsesAtomicUpsert 验证写入只有一条原子 Upsert，不存在 SELECT 后更新窗口。
func TestAddClicksUsesAtomicUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	bucket := time.Date(2026, 8, 15, 3, 27, 0, 0, time.UTC)
	query := regexp.QuoteMeta(`
		INSERT INTO button_click_minutes (user_id, button_key, minute_bucket, click_count)
		VALUES (?, ?, ?, ?) AS incoming
		ON DUPLICATE KEY UPDATE click_count = button_click_minutes.click_count + incoming.click_count
	`)
	mock.ExpectExec(query).
		WithArgs(int64(1), "yanlili", bucket, int64(5)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewButtonClickRepository(db)
	if err := repo.AddClicks(context.Background(), 1, "yanlili", bucket, 5); err != nil {
		t.Fatalf("AddClicks() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

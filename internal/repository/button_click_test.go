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
		INSERT INTO button_click_minutes (user_id, companion_binding_id, target_user_id, button_key, minute_bucket, click_count)
		SELECT membership.user_id, membership.binding_id, membership.partner_user_id, ?, ?, ?
		FROM companion_active_memberships AS membership
		WHERE membership.user_id = ? AND membership.binding_id = ? AND membership.partner_user_id = ?
		ON DUPLICATE KEY UPDATE click_count = button_click_minutes.click_count + ?
	`)
	mock.ExpectExec(query).
		WithArgs("yanlili", bucket, int64(5), int64(1), int64(9), int64(2), int64(5)).
		WillReturnResult(sqlmock.NewResult(1, 1))

	repo := NewButtonClickRepository(db)
	if err := repo.AddClicks(context.Background(), 9, 1, 2, "yanlili", bucket, 5); err != nil {
		t.Fatalf("AddClicks() error = %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

// TestDetailedRecordsMergeBindingInstances 验证详细记录按用户对和分钟合并，不返回绑定实例。
func TestDetailedRecordsMergeBindingInstances(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT.*COUNT\\(DISTINCT minute_bucket\\)").
		WithArgs(int64(3), int64(4), "yanlili", int64(4), int64(3), "yanlili").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(2)))
	mock.ExpectQuery("SELECT sort_id, direction, minute_bucket, click_count").
		WithArgs(int64(3), int64(4), "yanlili", int64(4), int64(3), "yanlili", 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"sort_id", "direction", "minute_bucket", "click_count"}).
			AddRow(int64(9), "mine", time.Date(2026, 8, 15, 10, 37, 0, 0, time.UTC), int64(2508)).
			AddRow(int64(10), "theirs", time.Date(2026, 8, 15, 10, 36, 0, 0, time.UTC), int64(2)))

	page, err := NewButtonClickRepository(db).GetDetailedRecords(context.Background(), 3, 4, "yanlili", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if page.TotalItems != 2 || len(page.Items) != 2 || page.Items[0].Direction != "mine" || page.Items[0].Count != 2508 {
		t.Fatalf("详细记录分页 = %#v", page)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

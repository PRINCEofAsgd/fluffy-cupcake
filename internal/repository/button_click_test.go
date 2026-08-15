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

// TestPairStatsIgnoreBindingInstance 验证当前摘要只按用户对过滤，并在分钟层合并历次绑定行。
func TestPairStatsIgnoreBindingInstance(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	mock.ExpectQuery("SELECT COALESCE\\(SUM\\(click_count\\), 0\\)").
		WithArgs(int64(3), int64(4), "yanlili").
		WillReturnRows(sqlmock.NewRows([]string{"total"}).AddRow(int64(2509)))
	mock.ExpectQuery("SELECT DATE_FORMAT\\(DATE_ADD\\(minute_bucket").
		WithArgs(480, int64(3), int64(4), "yanlili", 8).
		WillReturnRows(sqlmock.NewRows([]string{"date", "count"}).AddRow("2026-08-15", int64(2509)))
	mock.ExpectQuery("SELECT minute_bucket, SUM\\(click_count\\)").
		WithArgs(int64(3), int64(4), "yanlili", 8).
		WillReturnRows(sqlmock.NewRows([]string{"minute", "count"}).
			AddRow(time.Date(2026, 8, 15, 16, 3, 0, 0, time.UTC), int64(2)))

	repo := NewButtonClickRepository(db)
	total, err := repo.GetTotalCount(context.Background(), 3, 4, "yanlili")
	if err != nil || total != 2509 {
		t.Fatalf("total=%d error=%v", total, err)
	}
	daily, err := repo.GetDailyCounts(context.Background(), 3, 4, "yanlili", 480, 8)
	if err != nil || len(daily) != 1 || daily[0].Count != 2509 {
		t.Fatalf("daily=%#v error=%v", daily, err)
	}
	minutes, err := repo.GetClickMinutes(context.Background(), 3, 4, "yanlili", 8)
	if err != nil || len(minutes) != 1 || minutes[0].Count != 2 {
		t.Fatalf("minutes=%#v error=%v", minutes, err)
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

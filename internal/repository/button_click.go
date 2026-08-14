package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

// ButtonClickRepository 负责分钟桶原子累加及共享按钮统计查询。
type ButtonClickRepository struct {
	db *sql.DB
}

// NewButtonClickRepository 创建按钮点击数据访问对象。
func NewButtonClickRepository(db *sql.DB) *ButtonClickRepository {
	return &ButtonClickRepository{db: db}
}

// AddClicks 使用 MySQL 行别名 Upsert 原子累加同一用户、按钮和分钟桶的点击数。
func (r *ButtonClickRepository) AddClicks(ctx context.Context, userID int64, buttonKey string, minuteBucket time.Time, count int64) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO button_click_minutes (user_id, button_key, minute_bucket, click_count)
		VALUES (?, ?, ?, ?) AS incoming
		ON DUPLICATE KEY UPDATE click_count = button_click_minutes.click_count + incoming.click_count
	`, userID, buttonKey, minuteBucket.UTC(), count)
	if err != nil {
		return fmt.Errorf("累加按钮点击: %w", err)
	}
	return nil
}

// GetTotalCount 统计指定按钮跨所有用户和分钟的总点击数。
func (r *ButtonClickRepository) GetTotalCount(ctx context.Context, buttonKey string) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(click_count), 0)
		FROM button_click_minutes
		WHERE button_key = ?
	`, buttonKey).Scan(&total); err != nil {
		return 0, fmt.Errorf("查询按钮总点击数: %w", err)
	}
	return total, nil
}

// GetDailyCounts 按 UTC 日期汇总指定按钮的所有用户点击数。
func (r *ButtonClickRepository) GetDailyCounts(ctx context.Context, buttonKey string) ([]model.DailyClickCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DATE_FORMAT(minute_bucket, '%Y-%m-%d') AS click_date, SUM(click_count)
		FROM button_click_minutes
		WHERE button_key = ?
		GROUP BY click_date
		ORDER BY click_date
	`, buttonKey)
	if err != nil {
		return nil, fmt.Errorf("查询每日点击数: %w", err)
	}
	defer rows.Close()

	counts := make([]model.DailyClickCount, 0)
	for rows.Next() {
		var item model.DailyClickCount
		if err := rows.Scan(&item.Date, &item.Count); err != nil {
			return nil, fmt.Errorf("读取每日点击数: %w", err)
		}
		counts = append(counts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历每日点击数: %w", err)
	}
	return counts, nil
}

// GetClickMinutes 按分钟合并不同用户记录，保证时间列表每分钟只有一项。
func (r *ButtonClickRepository) GetClickMinutes(ctx context.Context, buttonKey string) ([]model.MinuteClickCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT minute_bucket, SUM(click_count)
		FROM button_click_minutes
		WHERE button_key = ?
		GROUP BY minute_bucket
		ORDER BY minute_bucket
	`, buttonKey)
	if err != nil {
		return nil, fmt.Errorf("查询分钟点击数: %w", err)
	}
	defer rows.Close()

	counts := make([]model.MinuteClickCount, 0)
	for rows.Next() {
		var item model.MinuteClickCount
		if err := rows.Scan(&item.Time, &item.Count); err != nil {
			return nil, fmt.Errorf("读取分钟点击数: %w", err)
		}
		item.Time = item.Time.UTC()
		counts = append(counts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历分钟点击数: %w", err)
	}
	return counts, nil
}

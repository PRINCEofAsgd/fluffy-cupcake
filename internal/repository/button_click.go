package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

// ButtonClickRepository 负责关系方向内的分钟桶原子累加、限量统计和详细分页查询。
type ButtonClickRepository struct {
	db *sql.DB
}

// NewButtonClickRepository 创建按钮点击数据访问对象。
func NewButtonClickRepository(db *sql.DB) *ButtonClickRepository {
	return &ButtonClickRepository{db: db}
}

// AddClicks 使用关系、发起人、对象、按钮和分钟桶联合唯一键原子累加。
func (r *ButtonClickRepository) AddClicks(ctx context.Context, bindingID, userID, targetUserID int64, buttonKey string, minuteBucket time.Time, count int64) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO button_click_minutes (user_id, companion_binding_id, target_user_id, button_key, minute_bucket, click_count)
		SELECT membership.user_id, membership.binding_id, membership.partner_user_id, ?, ?, ?
		FROM companion_active_memberships AS membership
		WHERE membership.user_id = ? AND membership.binding_id = ? AND membership.partner_user_id = ?
		ON DUPLICATE KEY UPDATE click_count = button_click_minutes.click_count + ?
	`, buttonKey, minuteBucket.UTC(), count, userID, bindingID, targetUserID, count)
	if err != nil {
		return fmt.Errorf("累加想念点击: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("读取想念点击结果: %w", err)
	}
	if changed == 0 {
		return ErrInvalidBindingState
	}
	return nil
}

// GetTotalCount 按用户对统计指定方向的历次总数，不以某一次绑定 ID 截断历史。
func (r *ButtonClickRepository) GetTotalCount(ctx context.Context, userID, targetUserID int64, buttonKey string) (int64, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(click_count), 0)
		FROM button_click_minutes
		WHERE user_id = ? AND target_user_id = ? AND button_key = ?
	`, userID, targetUserID, buttonKey).Scan(&total); err != nil {
		return 0, fmt.Errorf("查询方向想念总数: %w", err)
	}
	return total, nil
}

// GetDailyCounts 按浏览器 UTC 偏移生成本地日期，跨历次绑定聚合后只返回最新若干条。
func (r *ButtonClickRepository) GetDailyCounts(ctx context.Context, userID, targetUserID int64, buttonKey string, utcOffsetMinutes, limit int) ([]model.DailyClickCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT DATE_FORMAT(DATE_ADD(minute_bucket, INTERVAL ? MINUTE), '%Y-%m-%d') AS local_date, SUM(click_count)
		FROM button_click_minutes
		WHERE user_id = ? AND target_user_id = ? AND button_key = ?
		GROUP BY local_date
		ORDER BY local_date DESC
		LIMIT ?
	`, utcOffsetMinutes, userID, targetUserID, buttonKey, limit)
	if err != nil {
		return nil, fmt.Errorf("查询最近每日想念: %w", err)
	}
	defer rows.Close()
	counts := make([]model.DailyClickCount, 0, limit)
	for rows.Next() {
		var item model.DailyClickCount
		if err := rows.Scan(&item.Date, &item.Count); err != nil {
			return nil, fmt.Errorf("读取每日想念: %w", err)
		}
		counts = append(counts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历每日想念: %w", err)
	}
	return counts, nil
}

// GetClickMinutes 按用户对合并同分钟的迁移旧行和历次绑定行，再读取最新若干分钟桶。
func (r *ButtonClickRepository) GetClickMinutes(ctx context.Context, userID, targetUserID int64, buttonKey string, limit int) ([]model.MinuteClickCount, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT minute_bucket, SUM(click_count) AS click_count
		FROM button_click_minutes
		WHERE user_id = ? AND target_user_id = ? AND button_key = ?
		GROUP BY minute_bucket
		ORDER BY minute_bucket DESC
		LIMIT ?
	`, userID, targetUserID, buttonKey, limit)
	if err != nil {
		return nil, fmt.Errorf("查询最近想念分钟: %w", err)
	}
	defer rows.Close()
	counts := make([]model.MinuteClickCount, 0, limit)
	for rows.Next() {
		var item model.MinuteClickCount
		if err := rows.Scan(&item.Time, &item.Count); err != nil {
			return nil, fmt.Errorf("读取最近想念分钟: %w", err)
		}
		item.Time = item.Time.UTC()
		counts = append(counts, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历最近想念分钟: %w", err)
	}
	return counts, nil
}

// GetDetailedRecords 按用户对合并所有历次绑定和迁移旧数据，不向页面暴露绑定实例。
func (r *ButtonClickRepository) GetDetailedRecords(ctx context.Context, userID, partnerID int64, buttonKey string, page, pageSize int) (model.DetailedClickPage, error) {
	var total int64
	if err := r.db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(DISTINCT minute_bucket) FROM button_click_minutes WHERE user_id = ? AND target_user_id = ? AND button_key = ?)
			+
			(SELECT COUNT(DISTINCT minute_bucket) FROM button_click_minutes WHERE user_id = ? AND target_user_id = ? AND button_key = ?)
	`, userID, partnerID, buttonKey, partnerID, userID, buttonKey).Scan(&total); err != nil {
		return model.DetailedClickPage{}, fmt.Errorf("统计详细想念记录: %w", err)
	}
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx, `
		SELECT sort_id, direction, minute_bucket, click_count
		FROM (
			SELECT MAX(id) AS sort_id, 'mine' AS direction, minute_bucket, SUM(click_count) AS click_count
			FROM button_click_minutes
			WHERE user_id = ? AND target_user_id = ? AND button_key = ?
			GROUP BY minute_bucket
			UNION ALL
			SELECT MAX(id) AS sort_id, 'theirs' AS direction, minute_bucket, SUM(click_count) AS click_count
			FROM button_click_minutes
			WHERE user_id = ? AND target_user_id = ? AND button_key = ?
			GROUP BY minute_bucket
		) AS details
		ORDER BY minute_bucket DESC, sort_id DESC
		LIMIT ? OFFSET ?
	`, userID, partnerID, buttonKey, partnerID, userID, buttonKey, pageSize, offset)
	if err != nil {
		return model.DetailedClickPage{}, fmt.Errorf("查询详细想念记录: %w", err)
	}
	defer rows.Close()
	items := make([]model.DetailedClickRecord, 0, pageSize)
	for rows.Next() {
		var item model.DetailedClickRecord
		var sortID int64
		if err := rows.Scan(&sortID, &item.Direction, &item.Time, &item.Count); err != nil {
			return model.DetailedClickPage{}, fmt.Errorf("读取详细想念记录: %w", err)
		}
		item.Time = item.Time.UTC()
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return model.DetailedClickPage{}, fmt.Errorf("遍历详细想念记录: %w", err)
	}
	totalPages := 0
	if total > 0 {
		totalPages = int((total + int64(pageSize) - 1) / int64(pageSize))
	}
	return model.DetailedClickPage{Items: items, Page: page, PageSize: pageSize, TotalItems: total, TotalPages: totalPages}, nil
}

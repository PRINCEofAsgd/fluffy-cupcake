package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

const (
	// YanliliButtonKey 是当前共享按钮稳定的业务标识，与具体 DOM 元素解耦。
	YanliliButtonKey = "yanlili"
	// MaxClickDelta 限制一次批量请求可累计的点击数，避免明显的数据污染。
	MaxClickDelta = 100
)

var ErrInvalidClickCount = errors.New("count 必须在 1 到 100 之间")

// ButtonClickStore 描述点击服务需要的原子写入和统计查询能力。
type ButtonClickStore interface {
	AddClicks(ctx context.Context, userID int64, buttonKey string, minuteBucket time.Time, count int64) error
	GetTotalCount(ctx context.Context, buttonKey string) (int64, error)
	GetDailyCounts(ctx context.Context, buttonKey string) ([]model.DailyClickCount, error)
	GetClickMinutes(ctx context.Context, buttonKey string) ([]model.MinuteClickCount, error)
}

// ButtonClickService 负责可信用户、UTC 分钟桶和共享统计业务规则。
type ButtonClickService struct {
	clicks ButtonClickStore
	now    func() time.Time
}

// NewButtonClickService 创建按钮点击业务服务。
func NewButtonClickService(clicks ButtonClickStore) *ButtonClickService {
	return &ButtonClickService{clicks: clicks, now: time.Now}
}

// AddClicks 将服务端当前 UTC 时间向下截断到分钟，再原子累加 delta。
func (s *ButtonClickService) AddClicks(ctx context.Context, userID int64, count int64) error {
	if userID <= 0 {
		return ErrInvalidToken
	}
	if count <= 0 || count > MaxClickDelta {
		return ErrInvalidClickCount
	}
	bucket := s.now().UTC().Truncate(time.Minute)
	if err := s.clicks.AddClicks(ctx, userID, YanliliButtonKey, bucket, count); err != nil {
		return fmt.Errorf("记录按钮点击: %w", err)
	}
	return nil
}

// Stats 查询指定共享按钮的总数、UTC 每日数和聚合分钟列表。
func (s *ButtonClickService) Stats(ctx context.Context) (model.ButtonClickStats, error) {
	total, err := s.clicks.GetTotalCount(ctx, YanliliButtonKey)
	if err != nil {
		return model.ButtonClickStats{}, fmt.Errorf("读取总点击数: %w", err)
	}
	daily, err := s.clicks.GetDailyCounts(ctx, YanliliButtonKey)
	if err != nil {
		return model.ButtonClickStats{}, fmt.Errorf("读取每日点击数: %w", err)
	}
	minutes, err := s.clicks.GetClickMinutes(ctx, YanliliButtonKey)
	if err != nil {
		return model.ButtonClickStats{}, fmt.Errorf("读取分钟点击数: %w", err)
	}
	return model.ButtonClickStats{TotalCount: total, DailyCounts: daily, ClickMinutes: minutes}, nil
}

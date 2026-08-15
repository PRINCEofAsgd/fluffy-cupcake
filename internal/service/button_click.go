package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
)

const (
	// YanliliButtonKey 是当前想念按钮稳定的业务标识。
	YanliliButtonKey = "yanlili"
	// MaxClickDelta 限制一次批量写入的最大点击数。
	MaxClickDelta = 100
	// RecentStatsLimit 是两个当前摘要各自的数据库查询上限。
	RecentStatsLimit = 8
	// DetailedRecordsPageSize 是详细记录固定分页大小。
	DetailedRecordsPageSize = 20
)

var ErrInvalidClickCount = errors.New("count 必须在 1 到 100 之间")

// ButtonClickStore 描述点击服务需要的关系方向写入、限量统计和详细分页能力。
type ButtonClickStore interface {
	AddClicks(ctx context.Context, bindingID, userID, targetUserID int64, buttonKey string, minuteBucket time.Time, count int64) error
	GetTotalCount(ctx context.Context, bindingID, userID, targetUserID int64, buttonKey string) (int64, error)
	GetDailyCounts(ctx context.Context, bindingID, userID, targetUserID int64, buttonKey string, limit int) ([]model.DailyClickCount, error)
	GetClickMinutes(ctx context.Context, bindingID, userID, targetUserID int64, buttonKey string, limit int) ([]model.MinuteClickCount, error)
	GetDetailedRecords(ctx context.Context, userID, partnerID int64, buttonKey string, page, pageSize int) (model.DetailedClickPage, error)
}

// ActiveCompanionStore 描述点击服务解析当前绑定对象所需的能力。
type ActiveCompanionStore interface {
	GetActiveState(ctx context.Context, userID int64) (model.CompanionState, error)
}

// ButtonClickService 负责可信用户、活跃绑定方向、UTC 分钟桶和分页规则。
type ButtonClickService struct {
	clicks   ButtonClickStore
	bindings ActiveCompanionStore
	now      func() time.Time
}

// NewButtonClickService 创建按钮点击业务服务。
func NewButtonClickService(clicks ButtonClickStore, bindings ActiveCompanionStore) *ButtonClickService {
	return &ButtonClickService{clicks: clicks, bindings: bindings, now: time.Now}
}

// AddClicks 仅在双向绑定存在时，把点击记录到当前用户指向绑定对象的分钟桶。
func (s *ButtonClickService) AddClicks(ctx context.Context, userID int64, count int64) error {
	if userID <= 0 {
		return ErrInvalidToken
	}
	if count <= 0 || count > MaxClickDelta {
		return ErrInvalidClickCount
	}
	state, err := s.bindings.GetActiveState(ctx, userID)
	if err != nil {
		return fmt.Errorf("读取点击绑定对象: %w", err)
	}
	if !state.Bound {
		return ErrNotBound
	}
	bucket := s.now().UTC().Truncate(time.Minute)
	if err := s.clicks.AddClicks(ctx, state.BindingID, userID, state.PartnerID, YanliliButtonKey, bucket, count); err != nil {
		if errors.Is(err, repository.ErrInvalidBindingState) {
			return ErrNotBound
		}
		return fmt.Errorf("记录想念点击: %w", err)
	}
	return nil
}

// Stats 查询当前绑定“我想 ta”或“ta 想我”的总数和各自最新 8 条统计。
func (s *ButtonClickService) Stats(ctx context.Context, userID int64, direction string) (model.ButtonClickStats, error) {
	if direction != "mine" && direction != "theirs" {
		return model.ButtonClickStats{}, ErrInvalidDirection
	}
	state, err := s.bindings.GetActiveState(ctx, userID)
	if err != nil {
		return model.ButtonClickStats{}, fmt.Errorf("读取统计绑定对象: %w", err)
	}
	if !state.Bound {
		return model.ButtonClickStats{}, ErrNotBound
	}
	actorID, targetID := userID, state.PartnerID
	if direction == "theirs" {
		actorID, targetID = state.PartnerID, userID
	}
	total, err := s.clicks.GetTotalCount(ctx, state.BindingID, actorID, targetID, YanliliButtonKey)
	if err != nil {
		return model.ButtonClickStats{}, fmt.Errorf("读取想念总数: %w", err)
	}
	daily, err := s.clicks.GetDailyCounts(ctx, state.BindingID, actorID, targetID, YanliliButtonKey, RecentStatsLimit)
	if err != nil {
		return model.ButtonClickStats{}, fmt.Errorf("读取每日想念: %w", err)
	}
	minutes, err := s.clicks.GetClickMinutes(ctx, state.BindingID, actorID, targetID, YanliliButtonKey, RecentStatsLimit)
	if err != nil {
		return model.ButtonClickStats{}, fmt.Errorf("读取最近想念: %w", err)
	}
	return model.ButtonClickStats{TotalCount: total, DailyCounts: daily, ClickMinutes: minutes}, nil
}

// Details 查询与指定历次对象的全部双向分钟记录，页码从 1 开始且每页固定 20 条。
func (s *ButtonClickService) Details(ctx context.Context, userID, partnerID int64, page int) (model.DetailedClickPage, error) {
	if userID <= 0 || partnerID <= 0 || userID == partnerID {
		return model.DetailedClickPage{}, ErrCannotBindSelf
	}
	if page <= 0 {
		page = 1
	}
	return s.clicks.GetDetailedRecords(ctx, userID, partnerID, YanliliButtonKey, page, DetailedRecordsPageSize)
}

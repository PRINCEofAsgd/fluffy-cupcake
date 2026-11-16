package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
)

// maxMoodCustomLength 限制每组自定义文字的 Unicode 字符数。
const maxMoodCustomLength = 5

var (
	// ErrInvalidMood 表示心情字段不符合公开输入规则。
	ErrInvalidMood = errors.New("心情内容无效")
	// ErrMoodBindingChanged 表示编辑所用的绑定已经结束或更换。
	ErrMoodBindingChanged = errors.New("当前陪伴绑定已变更，请重新读取心情")

	moodStatuses = []string{"happy", "relaxed", "tired", "exhausted", "angry", "disappointed", "normal", "hungry"}
	moodReasons  = []string{"you", "family", "friends", "work", "health", "weather", "food", "bad_luck", "good_luck", "games"}
	moodComforts = []string{"company", "talk", "see_you", "milk_tea", "tasty_food", "go_out"}
)

// CompanionMoodStore 描述仅限当前绑定的心情读取和本人保存能力。
type CompanionMoodStore interface {
	GetCurrent(ctx context.Context, userID int64, theirs bool) (model.CompanionMoodView, error)
	SaveCurrent(ctx context.Context, userID int64, view model.CompanionMoodView) error
}

// CompanionMoodService 负责心情视角、固定选项和自定义文字校验。
type CompanionMoodService struct {
	moods CompanionMoodStore
}

// NewCompanionMoodService 创建陪伴心情服务。
func NewCompanionMoodService(moods CompanionMoodStore) *CompanionMoodService {
	return &CompanionMoodService{moods: moods}
}

// Get 从当前绑定读取对方或自己的心情。
func (s *CompanionMoodService) Get(ctx context.Context, userID int64, direction string) (model.CompanionMoodView, error) {
	if direction != "mine" && direction != "theirs" {
		return model.CompanionMoodView{}, ErrInvalidDirection
	}

	view, err := s.moods.GetCurrent(ctx, userID, direction == "theirs")
	if errors.Is(err, repository.ErrInvalidBindingState) {
		return model.CompanionMoodView{}, ErrNotBound
	}

	return view, err
}

// Save 校验三组完整选项并仅用可信用户身份保存自己的心情。
func (s *CompanionMoodService) Save(ctx context.Context, userID int64, view model.CompanionMoodView) (model.CompanionMoodView, error) {
	if view.BindingID <= 0 {
		return model.CompanionMoodView{}, fmt.Errorf("%w，binding_id 必须为正整数", ErrInvalidMood)
	}

	for _, group := range []struct {
		label   string
		choices []string
		allowed []string
	}{
		{"心情状态", view.Statuses, moodStatuses},
		{"影响原因", view.Reasons, moodReasons},
		{"想要安慰", view.Comforts, moodComforts},
	} {
		if err := validateMoodChoices(group.label, group.choices, group.allowed); err != nil {
			return model.CompanionMoodView{}, err
		}
	}

	for _, field := range []struct {
		label string
		value *string
	}{
		{"影响原因", &view.ReasonCustom},
		{"想要安慰", &view.ComfortCustom},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if utf8.RuneCountInString(*field.value) > maxMoodCustomLength ||
			strings.ContainsFunc(*field.value, unicode.IsControl) || strings.ContainsAny(*field.value, "\u2028\u2029") {
			return model.CompanionMoodView{}, fmt.Errorf("%w，%s的自定义文字最多 %d 个字符且须为单行", ErrInvalidMood, field.label, maxMoodCustomLength)
		}
	}

	if err := s.moods.SaveCurrent(ctx, userID, view); err != nil {
		if errors.Is(err, repository.ErrInvalidBindingState) {
			return model.CompanionMoodView{}, ErrMoodBindingChanged
		}
		return model.CompanionMoodView{}, err
	}

	return view, nil
}

// validateMoodChoices 要求传入数组并拒绝未知或重复选项，空数组表示清空该组。
func validateMoodChoices(label string, choices, allowed []string) error {
	if choices == nil || len(choices) > len(allowed) {
		return fmt.Errorf("%w，%s必须为固定选项数组", ErrInvalidMood, label)
	}

	for i, choice := range choices {
		if !slices.Contains(allowed, choice) || slices.Contains(choices[:i], choice) {
			return fmt.Errorf("%w，%s包含未知或重复选项", ErrInvalidMood, label)
		}
	}
	return nil
}

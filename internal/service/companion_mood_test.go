package service

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/repository"
)

// moodStoreStub 记录业务服务传下去的写入，并提供指定的绑定查询错误。
type moodStoreStub struct {
	err    error
	userID int64
	theirs bool
	saved  model.CompanionMoodView
}

// GetCurrent 记录读取身份与方向。
func (s *moodStoreStub) GetCurrent(_ context.Context, userID int64, theirs bool) (model.CompanionMoodView, error) {
	s.userID, s.theirs = userID, theirs
	return model.CompanionMoodView{BindingID: 9}, s.err
}

// SaveCurrent 捕获已校验且归一化的请求。
func (s *moodStoreStub) SaveCurrent(_ context.Context, userID int64, view model.CompanionMoodView) error {
	s.userID, s.saved = userID, view
	return s.err
}

// TestCompanionMoodSaveValidation 覆盖全量多选、清空和各字段校验，非法输入不得进入存储层。
func TestCompanionMoodSaveValidation(t *testing.T) {
	valid := model.CompanionMoodView{BindingID: 9, CompanionMood: model.CompanionMood{
		Statuses:     []string{"happy", "relaxed", "tired", "exhausted", "angry", "disappointed", "normal", "hungry"},
		Reasons:      []string{"you", "family", "friends", "work", "health", "weather", "food", "bad_luck", "good_luck", "games"},
		Comforts:     []string{"company", "talk", "see_you", "milk_tea", "tasty_food", "go_out"},
		ReasonCustom: "  有点小紧张  ", ComfortCustom: " \t ",
	}}
	store := new(moodStoreStub)
	svc := NewCompanionMoodService(store)
	got, err := svc.Save(t.Context(), 7, valid)
	if err != nil || got.ReasonCustom != "有点小紧张" || got.ComfortCustom != "" || store.userID != 7 || !reflect.DeepEqual(store.saved, got) {
		t.Fatalf("保存或归一化结果错误: %+v, %+v, %v", got, store.saved, err)
	}
	clear := model.CompanionMoodView{BindingID: 9, CompanionMood: model.CompanionMood{
		Statuses: []string{}, Reasons: []string{}, Comforts: []string{},
	}}
	if _, err := svc.Save(t.Context(), 7, clear); err != nil {
		t.Fatalf("清空三组失败: %v", err)
	}

	for _, tt := range []struct {
		name string
		edit func(*model.CompanionMoodView)
	}{
		{"非法绑定", func(v *model.CompanionMoodView) { v.BindingID = 0 }},
		{"缺失数组", func(v *model.CompanionMoodView) { v.Statuses = nil }},
		{"重复状态", func(v *model.CompanionMoodView) { v.Statuses = []string{"happy", "happy"} }},
		{"非法状态", func(v *model.CompanionMoodView) { v.Statuses = []string{"好开心"} }},
		{"非法原因", func(v *model.CompanionMoodView) { v.Reasons = []string{"happy"} }},
		{"非法安慰", func(v *model.CompanionMoodView) { v.Comforts = []string{"food"} }},
		{"自定义原因超长", func(v *model.CompanionMoodView) { v.ReasonCustom = "今天有点紧张" }},
		{"自定义安慰超长", func(v *model.CompanionMoodView) { v.ComfortCustom = "今天想吃火锅" }},
		{"多行自定义文字", func(v *model.CompanionMoodView) { v.ComfortCustom = "下\u2028雨" }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := valid
			tt.edit(&input)
			store.userID = 0
			if _, err := svc.Save(t.Context(), 7, input); !errors.Is(err, ErrInvalidMood) || store.userID != 0 {
				t.Fatalf("非法输入进入存储或错误类型异常: %v", err)
			}
		})
	}
	store.err = repository.ErrInvalidBindingState
	if _, err := svc.Save(t.Context(), 7, valid); !errors.Is(err, ErrMoodBindingChanged) {
		t.Fatalf("应拒绝已变更的绑定: %v", err)
	}
}

// TestCompanionMoodGetDirection 验证视角解析与解除绑定后的读权限。
func TestCompanionMoodGetDirection(t *testing.T) {
	store := new(moodStoreStub)
	svc := NewCompanionMoodService(store)
	if _, err := svc.Get(t.Context(), 7, "arbitrary"); !errors.Is(err, ErrInvalidDirection) || store.userID != 0 {
		t.Fatalf("非法视角进入存储或错误类型异常: %v", err)
	}
	for _, direction := range []string{"theirs", "mine"} {
		view, err := svc.Get(t.Context(), 7, direction)
		if err != nil || view.BindingID != 9 || store.userID != 7 || store.theirs != (direction == "theirs") {
			t.Fatalf("%s 视角解析错误: %+v, %v", direction, view, err)
		}
	}
	store.err = repository.ErrInvalidBindingState
	if _, err := svc.Get(t.Context(), 7, "mine"); !errors.Is(err, ErrNotBound) {
		t.Fatalf("未绑定读权限错误: %v", err)
	}
}

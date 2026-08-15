package service

import (
	"context"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

type memoryBucketKey struct {
	bindingID int64
	userID    int64
	targetID  int64
	buttonKey string
	minute    time.Time
}

type memoryClickStore struct{ buckets map[memoryBucketKey]int64 }

func newMemoryClickStore() *memoryClickStore {
	return &memoryClickStore{buckets: make(map[memoryBucketKey]int64)}
}

func (s *memoryClickStore) AddClicks(_ context.Context, bindingID, userID, targetID int64, buttonKey string, minute time.Time, count int64) error {
	s.buckets[memoryBucketKey{bindingID: bindingID, userID: userID, targetID: targetID, buttonKey: buttonKey, minute: minute}] += count
	return nil
}

func (s *memoryClickStore) GetTotalCount(_ context.Context, bindingID, userID, targetID int64, buttonKey string) (int64, error) {
	var total int64
	for key, count := range s.buckets {
		if key.bindingID == bindingID && key.userID == userID && key.targetID == targetID && key.buttonKey == buttonKey {
			total += count
		}
	}
	return total, nil
}

func (s *memoryClickStore) GetDailyCounts(_ context.Context, bindingID, userID, targetID int64, buttonKey string, limit int) ([]model.DailyClickCount, error) {
	grouped := make(map[string]int64)
	for key, count := range s.buckets {
		if key.bindingID == bindingID && key.userID == userID && key.targetID == targetID && key.buttonKey == buttonKey {
			grouped[key.minute.UTC().Format(time.DateOnly)] += count
		}
	}
	dates := make([]string, 0, len(grouped))
	for date := range grouped {
		dates = append(dates, date)
	}
	sortStringsDescending(dates)
	if len(dates) > limit {
		dates = dates[:limit]
	}
	result := make([]model.DailyClickCount, 0, len(dates))
	for _, date := range dates {
		result = append(result, model.DailyClickCount{Date: date, Count: grouped[date]})
	}
	return result, nil
}

func (s *memoryClickStore) GetClickMinutes(_ context.Context, bindingID, userID, targetID int64, buttonKey string, limit int) ([]model.MinuteClickCount, error) {
	minutes := make([]model.MinuteClickCount, 0)
	for key, count := range s.buckets {
		if key.bindingID == bindingID && key.userID == userID && key.targetID == targetID && key.buttonKey == buttonKey {
			minutes = append(minutes, model.MinuteClickCount{Time: key.minute, Count: count})
		}
	}
	for i := 0; i < len(minutes); i++ {
		for j := i + 1; j < len(minutes); j++ {
			if minutes[j].Time.After(minutes[i].Time) {
				minutes[i], minutes[j] = minutes[j], minutes[i]
			}
		}
	}
	if len(minutes) > limit {
		minutes = minutes[:limit]
	}
	return minutes, nil
}

func (s *memoryClickStore) GetDetailedRecords(_ context.Context, userID, partnerID int64, buttonKey string, page, pageSize int) (model.DetailedClickPage, error) {
	return model.DetailedClickPage{Page: page, PageSize: pageSize}, nil
}

type fakeActiveBindings struct {
	states map[int64]model.CompanionState
}

func (f fakeActiveBindings) GetActiveState(_ context.Context, userID int64) (model.CompanionState, error) {
	return f.states[userID], nil
}

// TestButtonClickMinuteBucketsAndDirectionalStats 验证同分钟累加、关系方向隔离和最近统计倒序。
func TestButtonClickMinuteBucketsAndDirectionalStats(t *testing.T) {
	store := newMemoryClickStore()
	bindings := fakeActiveBindings{states: map[int64]model.CompanionState{
		1: {Bound: true, BindingID: 9, PartnerID: 2},
		2: {Bound: true, BindingID: 9, PartnerID: 1},
	}}
	clicks := NewButtonClickService(store, bindings)
	now := time.Date(2026, 8, 15, 23, 59, 18, 0, time.UTC)
	clicks.now = func() time.Time { return now }

	for _, change := range []struct{ userID, count int64 }{{1, 3}, {1, 5}, {2, 2}} {
		if err := clicks.AddClicks(context.Background(), change.userID, change.count); err != nil {
			t.Fatal(err)
		}
	}
	key := memoryBucketKey{bindingID: 9, userID: 1, targetID: 2, buttonKey: YanliliButtonKey, minute: now.Truncate(time.Minute)}
	if got := store.buckets[key]; got != 8 {
		t.Fatalf("同一方向同一分钟累计 = %d，期望 8", got)
	}

	now = now.Add(time.Minute)
	if err := clicks.AddClicks(context.Background(), 1, 4); err != nil {
		t.Fatal(err)
	}
	mine, err := clicks.Stats(context.Background(), 1, "mine")
	if err != nil {
		t.Fatal(err)
	}
	if mine.TotalCount != 12 || len(mine.DailyCounts) != 2 || mine.DailyCounts[0].Date != "2026-08-16" || mine.ClickMinutes[0].Count != 4 {
		t.Fatalf("我想 ta 统计 = %#v", mine)
	}
	theirs, err := clicks.Stats(context.Background(), 1, "theirs")
	if err != nil {
		t.Fatal(err)
	}
	if theirs.TotalCount != 2 {
		t.Fatalf("ta 想我总数 = %d，期望 2", theirs.TotalCount)
	}
}

// TestButtonClickRejectsUnboundAndInvalidCount 验证未绑定不落库，非法 delta 也被拒绝。
func TestButtonClickRejectsUnboundAndInvalidCount(t *testing.T) {
	clicks := NewButtonClickService(newMemoryClickStore(), fakeActiveBindings{states: map[int64]model.CompanionState{}})
	if err := clicks.AddClicks(context.Background(), 1, 1); err != ErrNotBound {
		t.Fatalf("未绑定 error=%v", err)
	}
	for _, count := range []int64{0, -1, MaxClickDelta + 1} {
		if err := clicks.AddClicks(context.Background(), 1, count); err != ErrInvalidClickCount {
			t.Fatalf("count=%d error=%v", count, err)
		}
	}
	if _, err := clicks.Stats(context.Background(), 1, "bad"); err != ErrInvalidDirection {
		t.Fatalf("direction error=%v", err)
	}
}

func sortStringsDescending(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] > values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

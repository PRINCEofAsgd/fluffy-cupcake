package service

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/model"
)

type memoryBucketKey struct {
	userID    int64
	buttonKey string
	minute    time.Time
}

type memoryClickStore struct {
	buckets map[memoryBucketKey]int64
}

func newMemoryClickStore() *memoryClickStore {
	return &memoryClickStore{buckets: make(map[memoryBucketKey]int64)}
}

func (s *memoryClickStore) AddClicks(_ context.Context, userID int64, buttonKey string, minute time.Time, count int64) error {
	s.buckets[memoryBucketKey{userID: userID, buttonKey: buttonKey, minute: minute}] += count
	return nil
}

func (s *memoryClickStore) GetTotalCount(_ context.Context, buttonKey string) (int64, error) {
	var total int64
	for key, count := range s.buckets {
		if key.buttonKey == buttonKey {
			total += count
		}
	}
	return total, nil
}

func (s *memoryClickStore) GetDailyCounts(_ context.Context, buttonKey string) ([]model.DailyClickCount, error) {
	grouped := make(map[string]int64)
	for key, count := range s.buckets {
		if key.buttonKey == buttonKey {
			grouped[key.minute.UTC().Format(time.DateOnly)] += count
		}
	}
	dates := make([]string, 0, len(grouped))
	for date := range grouped {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	result := make([]model.DailyClickCount, 0, len(dates))
	for _, date := range dates {
		result = append(result, model.DailyClickCount{Date: date, Count: grouped[date]})
	}
	return result, nil
}

func (s *memoryClickStore) GetClickMinutes(_ context.Context, buttonKey string) ([]model.MinuteClickCount, error) {
	grouped := make(map[time.Time]int64)
	for key, count := range s.buckets {
		if key.buttonKey == buttonKey {
			grouped[key.minute] += count
		}
	}
	minutes := make([]time.Time, 0, len(grouped))
	for minute := range grouped {
		minutes = append(minutes, minute)
	}
	sort.Slice(minutes, func(i, j int) bool { return minutes[i].Before(minutes[j]) })
	result := make([]model.MinuteClickCount, 0, len(minutes))
	for _, minute := range minutes {
		result = append(result, model.MinuteClickCount{Time: minute, Count: grouped[minute]})
	}
	return result, nil
}

// TestButtonClickMinuteBucketsAndStats 验证同分钟累加、跨分钟/日期和多用户共享聚合。
func TestButtonClickMinuteBucketsAndStats(t *testing.T) {
	store := newMemoryClickStore()
	clicks := NewButtonClickService(store)
	now := time.Date(2026, 8, 15, 23, 59, 18, 0, time.UTC)
	clicks.now = func() time.Time { return now }

	for _, change := range []struct {
		userID int64
		count  int64
	}{
		{userID: 1, count: 3},
		{userID: 1, count: 5},
		{userID: 2, count: 2},
	} {
		if err := clicks.AddClicks(context.Background(), change.userID, change.count); err != nil {
			t.Fatal(err)
		}
	}
	if got := store.buckets[memoryBucketKey{userID: 1, buttonKey: YanliliButtonKey, minute: now.Truncate(time.Minute)}]; got != 8 {
		t.Fatalf("同一用户同一分钟累计 = %d，期望 8", got)
	}

	now = now.Add(time.Minute)
	if err := clicks.AddClicks(context.Background(), 1, 4); err != nil {
		t.Fatal(err)
	}
	stats, err := clicks.Stats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalCount != 14 {
		t.Fatalf("TotalCount = %d，期望 14", stats.TotalCount)
	}
	if fmt.Sprint(stats.DailyCounts) != "[{2026-08-15 10} {2026-08-16 4}]" {
		t.Fatalf("DailyCounts = %#v", stats.DailyCounts)
	}
	if len(stats.ClickMinutes) != 2 || stats.ClickMinutes[0].Count != 10 || stats.ClickMinutes[1].Count != 4 {
		t.Fatalf("ClickMinutes = %#v", stats.ClickMinutes)
	}
	if len(store.buckets) != 3 {
		t.Fatalf("底层用户分钟桶数量 = %d，期望 3", len(store.buckets))
	}
}

// TestButtonClickRejectsInvalidCount 验证客户端不能写入非正数或过大 delta。
func TestButtonClickRejectsInvalidCount(t *testing.T) {
	clicks := NewButtonClickService(newMemoryClickStore())
	for _, count := range []int64{0, -1, MaxClickDelta + 1} {
		if err := clicks.AddClicks(context.Background(), 1, count); err != ErrInvalidClickCount {
			t.Fatalf("count=%d error=%v", count, err)
		}
	}
}

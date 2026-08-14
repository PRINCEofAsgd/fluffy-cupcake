package model

import "time"

// DailyClickCount 表示一个 UTC 自然日内的共享按钮点击总数。
type DailyClickCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// MinuteClickCount 表示一个 UTC 分钟桶内所有用户的点击总数。
type MinuteClickCount struct {
	Time  time.Time `json:"time"`
	Count int64     `json:"count"`
}

// ButtonClickStats 汇总共享按钮的总数、每日计数和分钟时间点。
type ButtonClickStats struct {
	TotalCount   int64              `json:"total_count"`
	DailyCounts  []DailyClickCount  `json:"daily_counts"`
	ClickMinutes []MinuteClickCount `json:"click_minutes"`
}

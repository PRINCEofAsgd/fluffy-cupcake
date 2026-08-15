package model

import "time"

// DailyClickCount 表示一个 UTC 自然日内指定方向的想念次数。
type DailyClickCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// MinuteClickCount 表示一个 UTC 分钟桶内指定方向的想念次数。
type MinuteClickCount struct {
	Time  time.Time `json:"time"`
	Count int64     `json:"count"`
}

// ButtonClickStats 汇总当前对象指定方向的历次总数、最近每日计数和最近分钟时间点。
type ButtonClickStats struct {
	TotalCount   int64              `json:"total_count"`
	DailyCounts  []DailyClickCount  `json:"daily_counts"`
	ClickMinutes []MinuteClickCount `json:"click_minutes"`
}

// DetailedClickRecord 表示与历史对象之间按方向和分钟合并后的记录，不暴露绑定实例。
type DetailedClickRecord struct {
	Direction string    `json:"direction"`
	Time      time.Time `json:"time"`
	Count     int64     `json:"count"`
}

// DetailedClickPage 是固定每页 20 条的详细记录分页响应。
type DetailedClickPage struct {
	Items      []DetailedClickRecord `json:"items"`
	Page       int                   `json:"page"`
	PageSize   int                   `json:"page_size"`
	TotalItems int64                 `json:"total_items"`
	TotalPages int                   `json:"total_pages"`
}

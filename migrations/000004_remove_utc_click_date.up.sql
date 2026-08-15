-- 每日摘要改为按浏览器 UTC 偏移动态划分本地日期，不再保存容易与页面本地时间冲突的 UTC 日期列。
ALTER TABLE button_click_minutes
    DROP INDEX idx_click_relation_date,
    DROP COLUMN click_date;

-- 回滚前先把同一用户/按钮/分钟在不同绑定中的行合并，恢复旧唯一键时不会冲突。
CREATE TEMPORARY TABLE rollback_click_minutes AS
SELECT MIN(id) AS keep_id, SUM(click_count) AS merged_count
FROM button_click_minutes
GROUP BY user_id, button_key, minute_bucket;

UPDATE button_click_minutes AS clicks
JOIN rollback_click_minutes AS merged ON merged.keep_id = clicks.id
SET clicks.click_count = merged.merged_count;

DELETE clicks
FROM button_click_minutes AS clicks
LEFT JOIN rollback_click_minutes AS merged ON merged.keep_id = clicks.id
WHERE merged.keep_id IS NULL;

DROP TEMPORARY TABLE rollback_click_minutes;

ALTER TABLE button_click_minutes
    DROP FOREIGN KEY fk_click_binding,
    DROP FOREIGN KEY fk_click_target_user,
    DROP FOREIGN KEY fk_click_user,
    DROP CHECK chk_click_target_semantics,
    DROP INDEX uk_click_relation_user_button_minute,
    DROP INDEX idx_click_relation_date,
    DROP INDEX idx_click_user_target_minute;

ALTER TABLE button_click_minutes
    DROP COLUMN click_date,
    DROP COLUMN target_user_id,
    DROP COLUMN companion_binding_id,
    ADD UNIQUE KEY uk_click_user_button_minute (user_id, button_key, minute_bucket),
    ADD KEY idx_click_button_minute (button_key, minute_bucket),
    ADD CONSTRAINT fk_click_user FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT;

DROP TABLE companion_active_memberships;
DROP TABLE companion_bindings;

ALTER TABLE users DROP COLUMN last_login_at;

CREATE TABLE button_click_minutes (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    button_key VARCHAR(64) NOT NULL,
    minute_bucket DATETIME NOT NULL,
    click_count BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uk_click_user_button_minute (user_id, button_key, minute_bucket),
    KEY idx_click_button_minute (button_key, minute_bucket),
    CONSTRAINT fk_click_user FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT chk_click_count_positive CHECK (click_count > 0),
    CONSTRAINT chk_minute_bucket_aligned CHECK (SECOND(minute_bucket) = 0)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

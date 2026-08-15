ALTER TABLE button_click_minutes
    ADD COLUMN click_date DATE GENERATED ALWAYS AS (DATE(minute_bucket)) STORED AFTER minute_bucket,
    ADD KEY idx_click_relation_date (companion_binding_id, user_id, target_user_id, button_key, click_date DESC);

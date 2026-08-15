ALTER TABLE users
    ADD COLUMN last_login_at TIMESTAMP NULL DEFAULT NULL AFTER password_hash;

CREATE TABLE companion_bindings (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    inviter_id BIGINT UNSIGNED NOT NULL,
    invitee_id BIGINT UNSIGNED NOT NULL,
    inviter_note VARCHAR(64) NOT NULL DEFAULT '',
    invitee_note VARCHAR(64) NOT NULL DEFAULT '',
    status VARCHAR(16) NOT NULL,
    accepted_at TIMESTAMP NULL DEFAULT NULL,
    unbind_requested_by BIGINT UNSIGNED NULL,
    unbind_requested_at TIMESTAMP NULL DEFAULT NULL,
    ended_at TIMESTAMP NULL DEFAULT NULL,
    ended_by BIGINT UNSIGNED NULL,
    ended_reason VARCHAR(24) NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_binding_inviter_created (inviter_id, created_at DESC),
    KEY idx_binding_invitee_created (invitee_id, created_at DESC),
    KEY idx_binding_status_created (status, created_at DESC),
    CONSTRAINT fk_binding_inviter FOREIGN KEY (inviter_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT fk_binding_invitee FOREIGN KEY (invitee_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT fk_binding_unbind_requester FOREIGN KEY (unbind_requested_by) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT fk_binding_ender FOREIGN KEY (ended_by) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT chk_binding_distinct_users CHECK (inviter_id <> invitee_id),
    CONSTRAINT chk_binding_status CHECK (status IN ('pending', 'active', 'ended', 'superseded')),
    CONSTRAINT chk_binding_end_reason CHECK (ended_reason IS NULL OR ended_reason IN ('mutual', 'inactive'))
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

-- 该表仅保存当前占位；一用户一行由主键从数据库层保证“最多一个活跃绑定”。
CREATE TABLE companion_active_memberships (
    user_id BIGINT UNSIGNED NOT NULL,
    binding_id BIGINT UNSIGNED NOT NULL,
    partner_user_id BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id),
    UNIQUE KEY uk_active_membership_binding_user (binding_id, user_id),
    CONSTRAINT fk_active_membership_user FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT fk_active_membership_partner FOREIGN KEY (partner_user_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT fk_active_membership_binding FOREIGN KEY (binding_id) REFERENCES companion_bindings (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    CONSTRAINT chk_active_membership_distinct_users CHECK (user_id <> partner_user_id)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

ALTER TABLE button_click_minutes
    DROP FOREIGN KEY fk_click_user,
    DROP INDEX uk_click_user_button_minute,
    DROP INDEX idx_click_button_minute;

ALTER TABLE button_click_minutes
    ADD COLUMN companion_binding_id BIGINT UNSIGNED NULL AFTER user_id,
    ADD COLUMN target_user_id BIGINT UNSIGNED NULL AFTER companion_binding_id,
    ADD COLUMN click_date DATE GENERATED ALWAYS AS (DATE(minute_bucket)) STORED AFTER minute_bucket,
    ADD UNIQUE KEY uk_click_relation_user_button_minute (companion_binding_id, user_id, target_user_id, button_key, minute_bucket),
    ADD KEY idx_click_relation_date (companion_binding_id, user_id, target_user_id, button_key, click_date DESC),
    ADD KEY idx_click_user_target_minute (user_id, target_user_id, button_key, minute_bucket DESC),
    ADD CONSTRAINT fk_click_user FOREIGN KEY (user_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT fk_click_target_user FOREIGN KEY (target_user_id) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT fk_click_binding FOREIGN KEY (companion_binding_id) REFERENCES companion_bindings (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT chk_click_target_semantics CHECK (
        (companion_binding_id IS NULL AND target_user_id IS NULL)
        OR (target_user_id IS NOT NULL AND user_id <> target_user_id)
    );

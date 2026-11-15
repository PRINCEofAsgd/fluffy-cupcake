-- 保存固定截止的登录会话和刷新凭证哈希，已轮换的哈希用于检测重用。
CREATE TABLE auth_sessions (
    id CHAR(22) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    created_at DATETIME(6) NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    revoked_at DATETIME(6) NULL DEFAULT NULL,
    PRIMARY KEY (id),
    KEY idx_auth_sessions_user (user_id),
    CONSTRAINT fk_auth_sessions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

CREATE TABLE auth_refresh_tokens (
    token_hash BINARY(32) NOT NULL,
    session_id CHAR(22) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
    rotated_at DATETIME(6) NULL DEFAULT NULL,
    PRIMARY KEY (token_hash),
    KEY idx_auth_refresh_tokens_session (session_id),
    CONSTRAINT fk_auth_refresh_tokens_session FOREIGN KEY (session_id) REFERENCES auth_sessions (id) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

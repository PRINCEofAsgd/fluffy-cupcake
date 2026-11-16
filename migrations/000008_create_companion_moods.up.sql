-- 每段关系内，每个填写者只保留一份心情快照。
CREATE TABLE companion_moods (
    binding_id BIGINT UNSIGNED NOT NULL,
    author_user_id BIGINT UNSIGNED NOT NULL,
    content JSON NOT NULL,
    created_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    updated_at TIMESTAMP(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6) ON UPDATE CURRENT_TIMESTAMP(6),
    PRIMARY KEY (binding_id, author_user_id),
    KEY idx_companion_moods_author (author_user_id),
    CONSTRAINT fk_companion_moods_binding FOREIGN KEY (binding_id) REFERENCES companion_bindings (id) ON DELETE RESTRICT,
    CONSTRAINT fk_companion_moods_author FOREIGN KEY (author_user_id) REFERENCES users (id) ON DELETE RESTRICT,
    CONSTRAINT chk_companion_moods_content CHECK (JSON_TYPE(content) = 'OBJECT')
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COLLATE = utf8mb4_0900_ai_ci;

ALTER TABLE companion_bindings
    ADD COLUMN unbind_status VARCHAR(16) NOT NULL DEFAULT 'none' AFTER unbind_requested_at,
    ADD COLUMN unbind_responded_by BIGINT UNSIGNED NULL AFTER unbind_status,
    ADD COLUMN unbind_responded_at TIMESTAMP NULL DEFAULT NULL AFTER unbind_responded_by,
    ADD CONSTRAINT fk_binding_unbind_responder FOREIGN KEY (unbind_responded_by) REFERENCES users (id) ON UPDATE RESTRICT ON DELETE RESTRICT,
    ADD CONSTRAINT chk_binding_unbind_status CHECK (unbind_status IN ('none', 'pending', 'cancelled', 'rejected', 'accepted', 'direct'));

-- 兼容升级前已经存在的待处理申请和已完成解绑，不伪造新的信件或关系。
UPDATE companion_bindings
SET unbind_status = CASE
        WHEN ended_reason = 'mutual' THEN 'accepted'
        WHEN ended_reason = 'inactive' THEN 'direct'
        WHEN status = 'active' AND unbind_requested_by IS NOT NULL THEN 'pending'
        ELSE 'none'
    END,
    unbind_responded_by = CASE
        WHEN ended_reason IN ('mutual', 'inactive') THEN ended_by
        ELSE NULL
    END,
    unbind_responded_at = CASE
        WHEN ended_reason IN ('mutual', 'inactive') THEN ended_at
        ELSE NULL
    END;

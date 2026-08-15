-- 回滚到旧程序前清除已取消或已拒绝的申请人，避免旧程序误判为仍待处理。
UPDATE companion_bindings
SET unbind_requested_by = NULL,
    unbind_requested_at = NULL
WHERE unbind_status IN ('cancelled', 'rejected');

ALTER TABLE companion_bindings
    DROP FOREIGN KEY fk_binding_unbind_responder,
    DROP CHECK chk_binding_unbind_status,
    DROP COLUMN unbind_responded_at,
    DROP COLUMN unbind_responded_by,
    DROP COLUMN unbind_status;

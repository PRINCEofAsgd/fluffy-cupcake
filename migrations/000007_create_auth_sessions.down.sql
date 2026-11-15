-- 先删除引用会话的凭证表，再删除会话表。
DROP TABLE auth_refresh_tokens;
DROP TABLE auth_sessions;

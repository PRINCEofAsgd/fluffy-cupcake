#!/usr/bin/env bash
# 只读检查跨区域迁移前后的 MySQL 表、数据量、结构与 UTC 时间边界。

set -euo pipefail

readonly EXPECTED_TABLES="schema_migrations users button_click_minutes companion_bindings companion_active_memberships qr_login_mappings"
temp_dir=""

usage() {
  printf '%s\n' \
    '用法：' \
    '  ./scripts/db-migration/verify.sh preflight OLD' \
    '  ./scripts/db-migration/verify.sh preflight NEW' \
    '  ./scripts/db-migration/verify.sh verify' \
    '' \
    '连接变量：OLD_DB_HOST/PORT/USER/NAME 与 NEW_DB_HOST/PORT/USER/NAME' \
    '可选 TLS：OLD_DB_SSL_MODE/SSL_CA 与 NEW_DB_SSL_MODE/SSL_CA' \
    '密码始终由终端隐藏读取，不接受命令行参数，也不会写入报告。'
}

die() {
  printf '错误：%s\n' "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "$temp_dir" && -d "$temp_dir" ]]; then
    rm -rf -- "$temp_dir"
  fi
}
trap cleanup EXIT INT TERM

require_command() {
  command -v "$1" >/dev/null 2>&1 || die "缺少命令：$1"
}

reject_control_characters() {
  local name="$1"
  local value="$2"
  [[ "$value" != *$'\n'* && "$value" != *$'\r'* ]] || die "$name 不能包含换行符"
}

# option_file_value 对 MySQL option file 的反斜杠和单引号转义，支持非十六进制密码。
option_file_value() {
  local value="$1"
  value="${value//\\/\\\\}"
  value="${value//\'/\\\'}"
  printf "'%s'" "$value"
}

# load_side 把 OLD/NEW 环境变量复制到统一变量，避免 eval 解析外部输入。
load_side() {
  local side="$1"
  case "$side" in
    OLD)
      db_host="${OLD_DB_HOST:-}"
      db_port="${OLD_DB_PORT:-3306}"
      db_user="${OLD_DB_USER:-}"
      db_name="${OLD_DB_NAME:-fluffy_cupcake}"
      db_ssl_mode="${OLD_DB_SSL_MODE:-}"
      db_ssl_ca="${OLD_DB_SSL_CA:-}"
      ;;
    NEW)
      db_host="${NEW_DB_HOST:-}"
      db_port="${NEW_DB_PORT:-3306}"
      db_user="${NEW_DB_USER:-}"
      db_name="${NEW_DB_NAME:-fluffy_cupcake}"
      db_ssl_mode="${NEW_DB_SSL_MODE:-}"
      db_ssl_ca="${NEW_DB_SSL_CA:-}"
      ;;
    *) die "数据库侧必须是 OLD 或 NEW" ;;
  esac

  [[ -n "$db_host" ]] || die "缺少 ${side}_DB_HOST"
  [[ "$db_port" =~ ^[0-9]+$ ]] || die "${side}_DB_PORT 必须是数字"
  [[ -n "$db_user" ]] || die "缺少 ${side}_DB_USER"
  [[ -n "$db_name" ]] || die "缺少 ${side}_DB_NAME"
  reject_control_characters "${side}_DB_HOST" "$db_host"
  reject_control_characters "${side}_DB_USER" "$db_user"
  reject_control_characters "${side}_DB_NAME" "$db_name"

  if [[ -n "$db_ssl_mode" ]]; then
    case "$db_ssl_mode" in
      DISABLED|PREFERRED|REQUIRED|VERIFY_CA|VERIFY_IDENTITY) ;;
      *) die "${side}_DB_SSL_MODE 不是 MySQL 8 客户端支持的模式" ;;
    esac
  fi
  if [[ -n "$db_ssl_ca" && ! -r "$db_ssl_ca" ]]; then
    die "${side}_DB_SSL_CA 不可读"
  fi
}

# create_client_config 用 0600 临时文件保存交互式密码，避免 Secret 出现在 ps 或 shell history。
create_client_config() {
  local side="$1"
  local config_path="$2"
  local password

  load_side "$side"
  printf '%s 数据库密码（输入不会显示）: ' "$side" >&2
  IFS= read -r -s password
  printf '\n' >&2
  [[ -n "$password" ]] || die "$side 数据库密码不能为空"
  reject_control_characters "$side 数据库密码" "$password"

  umask 077
  {
    printf '[client]\n'
    printf 'host=%s\n' "$(option_file_value "$db_host")"
    printf 'port=%s\n' "$db_port"
    printf 'user=%s\n' "$(option_file_value "$db_user")"
    printf 'password=%s\n' "$(option_file_value "$password")"
    printf 'database=%s\n' "$(option_file_value "$db_name")"
    printf 'default-character-set=utf8mb4\n'
    if [[ -n "$db_ssl_mode" ]]; then
      printf 'ssl-mode=%s\n' "$db_ssl_mode"
    fi
    if [[ -n "$db_ssl_ca" ]]; then
      printf 'ssl-ca=%s\n' "$(option_file_value "$db_ssl_ca")"
    fi
  } >"$config_path"
  chmod 600 "$config_path"
  unset password
}

# mysql_query 每次固定 UTC 会话，仅允许调用方传入仓库内定义的只读查询。
mysql_query() {
  local config_path="$1"
  local query="$2"
  mysql --defaults-extra-file="$config_path" --batch --raw --skip-column-names \
    --connect-timeout=5 --execute="SET SESSION time_zone = '+00:00'; ${query}"
}

generate_row_count_sql() {
  local config_path="$1"
  mysql_query "$config_path" \
    "SELECT CONCAT('SELECT ', QUOTE(TABLE_NAME), ', COUNT(*) FROM ',
                   CHAR(96), REPLACE(TABLE_NAME, CHAR(96), CONCAT(CHAR(96), CHAR(96))), CHAR(96), ';')
       FROM information_schema.TABLES
       WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'
       ORDER BY TABLE_NAME;"
}

generate_temporal_sql() {
  local config_path="$1"
  mysql_query "$config_path" \
    "SELECT CONCAT(
       'SELECT ', QUOTE(TABLE_NAME), ', ', QUOTE(COLUMN_NAME), ', COUNT(',
       CHAR(96), REPLACE(COLUMN_NAME, CHAR(96), CONCAT(CHAR(96), CHAR(96))), CHAR(96), '), ',
       'COALESCE(DATE_FORMAT(MIN(', CHAR(96), REPLACE(COLUMN_NAME, CHAR(96), CONCAT(CHAR(96), CHAR(96))), CHAR(96), '), ''%Y-%m-%dT%H:%i:%s.%fZ''), ''<NULL>''), ',
       'COALESCE(DATE_FORMAT(MAX(', CHAR(96), REPLACE(COLUMN_NAME, CHAR(96), CONCAT(CHAR(96), CHAR(96))), CHAR(96), '), ''%Y-%m-%dT%H:%i:%s.%fZ''), ''<NULL>'') ',
       'FROM ', CHAR(96), REPLACE(TABLE_NAME, CHAR(96), CONCAT(CHAR(96), CHAR(96))), CHAR(96), ';')
     FROM information_schema.COLUMNS
     WHERE TABLE_SCHEMA = DATABASE()
       AND DATA_TYPE IN ('date', 'datetime', 'timestamp')
     ORDER BY TABLE_NAME, ORDINAL_POSITION;"
}

run_generated_sql() {
  local config_path="$1"
  local sql_file="$2"
  {
    printf "SET SESSION time_zone = '+00:00';\n"
    sed -n '1,$p' "$sql_file"
  } | mysql --defaults-extra-file="$config_path" --batch --raw --skip-column-names --connect-timeout=5
}

# snapshot 生成确定排序的只读快照，供 OLD/NEW 做逐文件比较。
snapshot() {
  local config_path="$1"
  local output_dir="$2"
  local generated_sql="$output_dir/generated.sql"
  mkdir -p "$output_dir"

  mysql_query "$config_path" \
    "SELECT TABLE_NAME, ENGINE, TABLE_COLLATION
       FROM information_schema.TABLES
       WHERE TABLE_SCHEMA = DATABASE() AND TABLE_TYPE = 'BASE TABLE'
       ORDER BY TABLE_NAME;" >"$output_dir/tables.tsv"

  generate_row_count_sql "$config_path" >"$generated_sql"
  run_generated_sql "$config_path" "$generated_sql" >"$output_dir/row-counts.tsv"

  if awk -F $'\t' '$1 == "schema_migrations" { found=1 } END { exit !found }' "$output_dir/tables.tsv"; then
    mysql_query "$config_path" "SELECT version, dirty FROM schema_migrations ORDER BY version;" >"$output_dir/schema-migrations.tsv"
  else
    : >"$output_dir/schema-migrations.tsv"
  fi

  mysql_query "$config_path" \
    "SELECT TABLE_NAME, ORDINAL_POSITION, COLUMN_NAME, COLUMN_TYPE, IS_NULLABLE,
            COALESCE(COLUMN_DEFAULT, '<NULL>'), EXTRA,
            COALESCE(CHARACTER_SET_NAME, '<NULL>'), COALESCE(COLLATION_NAME, '<NULL>')
       FROM information_schema.COLUMNS
       WHERE TABLE_SCHEMA = DATABASE()
       ORDER BY TABLE_NAME, ORDINAL_POSITION;" >"$output_dir/columns.tsv"

  mysql_query "$config_path" \
    "SELECT TABLE_NAME, INDEX_NAME, NON_UNIQUE, SEQ_IN_INDEX, COLUMN_NAME,
            COALESCE(COLLATION, '<NULL>'), COALESCE(SUB_PART, 0), INDEX_TYPE
       FROM information_schema.STATISTICS
       WHERE TABLE_SCHEMA = DATABASE()
       ORDER BY TABLE_NAME, INDEX_NAME, SEQ_IN_INDEX;" >"$output_dir/indexes.tsv"

  mysql_query "$config_path" \
    "SELECT k.TABLE_NAME, k.CONSTRAINT_NAME, k.COLUMN_NAME,
            k.REFERENCED_TABLE_NAME, k.REFERENCED_COLUMN_NAME,
            r.UPDATE_RULE, r.DELETE_RULE
       FROM information_schema.KEY_COLUMN_USAGE k
       JOIN information_schema.REFERENTIAL_CONSTRAINTS r
         ON r.CONSTRAINT_SCHEMA = k.CONSTRAINT_SCHEMA
        AND r.TABLE_NAME = k.TABLE_NAME
        AND r.CONSTRAINT_NAME = k.CONSTRAINT_NAME
       WHERE k.CONSTRAINT_SCHEMA = DATABASE()
         AND k.REFERENCED_TABLE_NAME IS NOT NULL
       ORDER BY k.TABLE_NAME, k.CONSTRAINT_NAME, k.ORDINAL_POSITION;" >"$output_dir/foreign-keys.tsv"

  generate_temporal_sql "$config_path" >"$generated_sql"
  run_generated_sql "$config_path" "$generated_sql" >"$output_dir/temporal-stats.tsv"
  rm -f -- "$generated_sql"
}

print_preflight() {
  local side="$1"
  local config_path="$2"
  local snapshot_dir="$temp_dir/${side}-preflight"
  snapshot "$config_path" "$snapshot_dir"

  printf '\n[%s] 基础信息（UTC session）\n' "$side"
  mysql_query "$config_path" \
    "SELECT 'mysql_version', VERSION();
     SELECT 'database', DATABASE();
     SELECT 'session_time_zone', @@session.time_zone;
     SELECT 'global_time_zone', @@global.time_zone;"
  printf '\n[%s] SHOW TABLES\n' "$side"
  mysql_query "$config_path" 'SHOW TABLES;'
  printf '\n[%s] schema_migrations\n' "$side"
  if [[ -s "$snapshot_dir/schema-migrations.tsv" ]]; then
    sed -n '1,$p' "$snapshot_dir/schema-migrations.tsv"
  else
    printf '<不存在或为空>\n'
  fi
  printf '\n[%s] 全部表精确行数\n' "$side"
  sed -n '1,$p' "$snapshot_dir/row-counts.tsv"
  printf '\n[%s] 时间字段非空数与 UTC 最小/最大值\n' "$side"
  sed -n '1,$p' "$snapshot_dir/temporal-stats.tsv"
}

compare_file() {
  local filename="$1"
  if diff -u "$temp_dir/OLD/$filename" "$temp_dir/NEW/$filename" >/dev/null; then
    printf '[一致] %s\n' "$filename"
  else
    printf '[不一致] %s\n' "$filename" >&2
    diff -u "$temp_dir/OLD/$filename" "$temp_dir/NEW/$filename" >&2 || true
    verify_failed=1
  fi
}

verify_snapshots() {
  local old_config="$1"
  local new_config="$2"
  local table
  snapshot "$old_config" "$temp_dir/OLD"
  snapshot "$new_config" "$temp_dir/NEW"

  verify_failed=0
  for table in $EXPECTED_TABLES; do
    if ! awk -F $'\t' -v table="$table" '$1 == table { found=1 } END { exit !found }' "$temp_dir/OLD/tables.tsv"; then
      printf '[缺失] OLD 没有仓库预期表 %s\n' "$table" >&2
      verify_failed=1
    fi
    if ! awk -F $'\t' -v table="$table" '$1 == table { found=1 } END { exit !found }' "$temp_dir/NEW/tables.tsv"; then
      printf '[缺失] NEW 没有仓库预期表 %s\n' "$table" >&2
      verify_failed=1
    fi
  done

  compare_file tables.tsv
  compare_file row-counts.tsv
  compare_file schema-migrations.tsv
  compare_file columns.tsv
  compare_file indexes.tsv
  compare_file foreign-keys.tsv
  compare_file temporal-stats.tsv

  printf '\n关键业务表行数（OLD；NEW 必须完全一致）\n'
  awk -F $'\t' '$1 == "users" || $1 == "button_click_minutes" || $1 == "companion_bindings" || $1 == "companion_active_memberships" || $1 == "qr_login_mappings"' "$temp_dir/OLD/row-counts.tsv"

  if [[ "$verify_failed" -ne 0 ]]; then
    die 'OLD/NEW 验证失败；保持停写，不要切换 DATABASE_DSN'
  fi
  printf '\n验证通过：表、精确行数、Migration、列、PK/UNIQUE/普通索引、外键及 UTC 时间边界均一致。\n'
}

main() {
  local command="${1:-}"
  local side
  require_command mysql
  require_command diff
  temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/fluffy-cupcake-db-verify.XXXXXX")"
  chmod 700 "$temp_dir"

  case "$command" in
    preflight)
      side="$(printf '%s' "${2:-}" | tr '[:lower:]' '[:upper:]')"
      [[ "$side" == OLD || "$side" == NEW ]] || { usage; exit 2; }
      create_client_config "$side" "$temp_dir/${side}.cnf"
      print_preflight "$side" "$temp_dir/${side}.cnf"
      ;;
    verify)
      [[ "$#" -eq 1 ]] || { usage; exit 2; }
      create_client_config OLD "$temp_dir/OLD.cnf"
      create_client_config NEW "$temp_dir/NEW.cnf"
      verify_snapshots "$temp_dir/OLD.cnf" "$temp_dir/NEW.cnf"
      ;;
    *)
      usage
      exit 2
      ;;
  esac
}

main "$@"

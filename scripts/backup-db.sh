#!/usr/bin/env bash
# =============================================================================
# PostgreSQL backup script for Turnero
# Creates compressed daily backups with rotation:
#   - Daily: keep last 7
#   - Weekly (Sundays): keep last 4
#
# Usage:
#   ./scripts/backup-db.sh
#
# Prerequisites:
#   - pg_dump installed
#   - PGPASSWORD or .pgpass configured
#   - Backup directory exists and is writable
# =============================================================================

set -euo pipefail

# --- Configuration ---
# TODO: Update these for your production environment
DB_NAME="${TURNERO_DB_NAME:-turnero}"
DB_USER="${TURNERO_DB_USER:-turnero}"
DB_HOST="${TURNERO_DB_HOST:-localhost}"
DB_PORT="${TURNERO_DB_PORT:-5432}"

BACKUP_DIR="${TURNERO_BACKUP_DIR:-/var/backups/turnero}"
DAILY_RETAIN=7
WEEKLY_RETAIN=4

# --- Derived values ---
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
DAY_OF_WEEK=$(date +%u)  # 1=Monday, 7=Sunday
DAILY_DIR="${BACKUP_DIR}/daily"
WEEKLY_DIR="${BACKUP_DIR}/weekly"

# --- Ensure directories exist ---
mkdir -p "${DAILY_DIR}" "${WEEKLY_DIR}"

# --- Create backup ---
BACKUP_FILE="${DAILY_DIR}/turnero_${TIMESTAMP}.sql.gz"

echo "[$(date -Iseconds)] Starting backup: ${DB_NAME} → ${BACKUP_FILE}"

pg_dump \
    -h "${DB_HOST}" \
    -p "${DB_PORT}" \
    -U "${DB_USER}" \
    -d "${DB_NAME}" \
    --format=custom \
    --compress=6 \
    --no-owner \
    --no-privileges \
    -f "${BACKUP_FILE}"

BACKUP_SIZE=$(du -h "${BACKUP_FILE}" | cut -f1)
echo "[$(date -Iseconds)] Backup completed: ${BACKUP_SIZE}"

# --- Weekly copy (on Sundays) ---
if [ "${DAY_OF_WEEK}" -eq 7 ]; then
    WEEKLY_FILE="${WEEKLY_DIR}/turnero_weekly_${TIMESTAMP}.sql.gz"
    cp "${BACKUP_FILE}" "${WEEKLY_FILE}"
    echo "[$(date -Iseconds)] Weekly backup created: ${WEEKLY_FILE}"
fi

# --- Rotate daily backups (keep last N) ---
DAILY_COUNT=$(find "${DAILY_DIR}" -name "turnero_*.sql.gz" -type f | wc -l)
if [ "${DAILY_COUNT}" -gt "${DAILY_RETAIN}" ]; then
    DELETE_COUNT=$((DAILY_COUNT - DAILY_RETAIN))
    find "${DAILY_DIR}" -name "turnero_*.sql.gz" -type f -printf '%T+ %p\n' \
        | sort \
        | head -n "${DELETE_COUNT}" \
        | cut -d' ' -f2- \
        | xargs rm -f
    echo "[$(date -Iseconds)] Rotated daily backups: deleted ${DELETE_COUNT} old backups"
fi

# --- Rotate weekly backups (keep last N) ---
WEEKLY_COUNT=$(find "${WEEKLY_DIR}" -name "turnero_weekly_*.sql.gz" -type f | wc -l)
if [ "${WEEKLY_COUNT}" -gt "${WEEKLY_RETAIN}" ]; then
    DELETE_COUNT=$((WEEKLY_COUNT - WEEKLY_RETAIN))
    find "${WEEKLY_DIR}" -name "turnero_weekly_*.sql.gz" -type f -printf '%T+ %p\n' \
        | sort \
        | head -n "${DELETE_COUNT}" \
        | cut -d' ' -f2- \
        | xargs rm -f
    echo "[$(date -Iseconds)] Rotated weekly backups: deleted ${DELETE_COUNT} old backups"
fi

echo "[$(date -Iseconds)] Backup process completed successfully"

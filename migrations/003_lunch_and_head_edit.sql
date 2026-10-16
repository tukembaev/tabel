-- +goose Up

ALTER TABLE user_snapshots
    ADD COLUMN IF NOT EXISTS can_edit_timesheet BOOLEAN NOT NULL DEFAULT FALSE;

-- Keep one session per employee per day (no lunch cycles).
DELETE FROM attendance_sessions a
WHERE a.id IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY university_id, user_id, work_date
                   ORDER BY opened_at ASC, created_at ASC
               ) AS rn
        FROM attendance_sessions
    ) t
    WHERE t.rn > 1
);

DELETE FROM attendance_sessions_archive a
WHERE a.id IN (
    SELECT id FROM (
        SELECT id,
               ROW_NUMBER() OVER (
                   PARTITION BY university_id, user_id, work_date
                   ORDER BY opened_at ASC, created_at ASC
               ) AS rn
        FROM attendance_sessions_archive
    ) t
    WHERE t.rn > 1
);

ALTER TABLE attendance_sessions
    ADD COLUMN IF NOT EXISTS lunch_seconds INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'scan',
    ADD COLUMN IF NOT EXISTS edited_by UUID,
    ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';

ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_close_reason_check;
ALTER TABLE attendance_sessions
    ADD CONSTRAINT attendance_sessions_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff', 'head'));

ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_source_check;
ALTER TABLE attendance_sessions
    ADD CONSTRAINT attendance_sessions_source_check
    CHECK (source IN ('scan', 'head', 'system'));

CREATE UNIQUE INDEX IF NOT EXISTS uq_day_session_per_user
    ON attendance_sessions (university_id, user_id, work_date);

ALTER TABLE attendance_sessions_archive
    ADD COLUMN IF NOT EXISTS lunch_seconds INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'scan',
    ADD COLUMN IF NOT EXISTS edited_by UUID,
    ADD COLUMN IF NOT EXISTS edited_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS note TEXT NOT NULL DEFAULT '';

ALTER TABLE attendance_sessions_archive DROP CONSTRAINT IF EXISTS attendance_sessions_archive_close_reason_check;
ALTER TABLE attendance_sessions_archive
    ADD CONSTRAINT attendance_sessions_archive_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff', 'head'));

ALTER TABLE attendance_sessions_archive DROP CONSTRAINT IF EXISTS attendance_sessions_archive_source_check;
ALTER TABLE attendance_sessions_archive
    ADD CONSTRAINT attendance_sessions_archive_source_check
    CHECK (source IN ('scan', 'head', 'system'));

CREATE TABLE IF NOT EXISTS timesheet_edits (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    university_id TEXT NOT NULL DEFAULT 'kstu',
    department_id TEXT,
    editor_id     UUID NOT NULL,
    user_id       UUID NOT NULL,
    work_date     DATE NOT NULL,
    action        TEXT NOT NULL,
    note          TEXT NOT NULL DEFAULT '',
    payload       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_timesheet_edits_dept ON timesheet_edits (university_id, department_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_timesheet_edits_user ON timesheet_edits (university_id, user_id, work_date);

-- +goose Down

DROP INDEX IF EXISTS idx_timesheet_edits_user;
DROP INDEX IF EXISTS idx_timesheet_edits_dept;
DROP TABLE IF EXISTS timesheet_edits;

ALTER TABLE attendance_sessions_archive DROP CONSTRAINT IF EXISTS attendance_sessions_archive_source_check;
ALTER TABLE attendance_sessions_archive DROP CONSTRAINT IF EXISTS attendance_sessions_archive_close_reason_check;
ALTER TABLE attendance_sessions_archive
    ADD CONSTRAINT attendance_sessions_archive_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff'));
ALTER TABLE attendance_sessions_archive
    DROP COLUMN IF EXISTS lunch_seconds,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS edited_by,
    DROP COLUMN IF EXISTS edited_at,
    DROP COLUMN IF EXISTS note;

DROP INDEX IF EXISTS uq_day_session_per_user;

ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_source_check;
ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_close_reason_check;
ALTER TABLE attendance_sessions
    ADD CONSTRAINT attendance_sessions_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff'));
ALTER TABLE attendance_sessions
    DROP COLUMN IF EXISTS lunch_seconds,
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS edited_by,
    DROP COLUMN IF EXISTS edited_at,
    DROP COLUMN IF EXISTS note;

ALTER TABLE user_snapshots DROP COLUMN IF EXISTS can_edit_timesheet;

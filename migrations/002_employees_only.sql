-- +goose Up

DROP TABLE IF EXISTS student_reputations;

ALTER TABLE user_snapshots DROP COLUMN IF EXISTS has_student;

DROP INDEX IF EXISTS uq_day_open_per_user;
DROP INDEX IF EXISTS uq_day_session_per_user;
DROP INDEX IF EXISTS uq_lesson_per_user;
DROP INDEX IF EXISTS idx_sessions_lesson;
DROP INDEX IF EXISTS idx_sessions_open_closes;
DROP INDEX IF EXISTS idx_sessions_open_day;

DELETE FROM attendance_sessions
    WHERE mode = 'lesson' OR acting_as IN ('student', 'teacher');
DELETE FROM attendance_sessions_archive
    WHERE mode = 'lesson' OR acting_as IN ('student', 'teacher');

UPDATE attendance_sessions
    SET close_reason = 'auto_cutoff' WHERE close_reason = 'auto_lesson';
UPDATE attendance_sessions_archive
    SET close_reason = 'auto_cutoff' WHERE close_reason = 'auto_lesson';

ALTER TABLE attendance_sessions
    DROP COLUMN IF EXISTS acting_as,
    DROP COLUMN IF EXISTS mode,
    DROP COLUMN IF EXISTS lesson_id,
    DROP COLUMN IF EXISTS closes_at,
    DROP COLUMN IF EXISTS trust,
    DROP COLUMN IF EXISTS room_id,
    ADD COLUMN IF NOT EXISTS entry_building TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS entry_room TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS exit_building TEXT,
    ADD COLUMN IF NOT EXISTS exit_room TEXT;

ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_close_reason_check;
ALTER TABLE attendance_sessions
    ADD CONSTRAINT attendance_sessions_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff'));

CREATE UNIQUE INDEX uq_day_open_per_user
    ON attendance_sessions (university_id, user_id, work_date)
    WHERE status = 'open';

CREATE INDEX idx_sessions_open_day
    ON attendance_sessions (work_date)
    WHERE status = 'open';

ALTER TABLE attendance_sessions_archive
    DROP COLUMN IF EXISTS acting_as,
    DROP COLUMN IF EXISTS mode,
    DROP COLUMN IF EXISTS lesson_id,
    DROP COLUMN IF EXISTS closes_at,
    DROP COLUMN IF EXISTS trust,
    DROP COLUMN IF EXISTS room_id,
    ADD COLUMN IF NOT EXISTS entry_building TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS entry_room TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS exit_building TEXT,
    ADD COLUMN IF NOT EXISTS exit_room TEXT;

ALTER TABLE attendance_sessions_archive DROP CONSTRAINT IF EXISTS attendance_sessions_archive_close_reason_check;
ALTER TABLE attendance_sessions_archive DROP CONSTRAINT IF EXISTS attendance_sessions_close_reason_check;
ALTER TABLE attendance_sessions_archive
    ADD CONSTRAINT attendance_sessions_archive_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff'));

-- +goose Down

ALTER TABLE attendance_sessions_archive DROP CONSTRAINT IF EXISTS attendance_sessions_archive_close_reason_check;
ALTER TABLE attendance_sessions_archive
    ADD CONSTRAINT attendance_sessions_archive_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff', 'auto_lesson'));

ALTER TABLE attendance_sessions_archive
    DROP COLUMN IF EXISTS entry_building,
    DROP COLUMN IF EXISTS entry_room,
    DROP COLUMN IF EXISTS exit_building,
    DROP COLUMN IF EXISTS exit_room,
    ADD COLUMN IF NOT EXISTS acting_as TEXT NOT NULL DEFAULT 'staff',
    ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'day',
    ADD COLUMN IF NOT EXISTS lesson_id TEXT,
    ADD COLUMN IF NOT EXISTS closes_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS trust TEXT NOT NULL DEFAULT 'scanned',
    ADD COLUMN IF NOT EXISTS room_id TEXT;

DROP INDEX IF EXISTS uq_day_open_per_user;
DROP INDEX IF EXISTS idx_sessions_open_day;

ALTER TABLE attendance_sessions DROP CONSTRAINT IF EXISTS attendance_sessions_close_reason_check;
ALTER TABLE attendance_sessions
    ADD CONSTRAINT attendance_sessions_close_reason_check
    CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff', 'auto_lesson'));

ALTER TABLE attendance_sessions
    DROP COLUMN IF EXISTS entry_building,
    DROP COLUMN IF EXISTS entry_room,
    DROP COLUMN IF EXISTS exit_building,
    DROP COLUMN IF EXISTS exit_room,
    ADD COLUMN IF NOT EXISTS acting_as TEXT NOT NULL DEFAULT 'staff',
    ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'day',
    ADD COLUMN IF NOT EXISTS lesson_id TEXT,
    ADD COLUMN IF NOT EXISTS closes_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS trust TEXT NOT NULL DEFAULT 'scanned',
    ADD COLUMN IF NOT EXISTS room_id TEXT;

CREATE UNIQUE INDEX uq_day_open_per_user
    ON attendance_sessions (university_id, user_id, work_date)
    WHERE mode = 'day' AND status = 'open';

CREATE UNIQUE INDEX uq_day_session_per_user
    ON attendance_sessions (university_id, user_id, work_date)
    WHERE mode = 'day';

CREATE UNIQUE INDEX uq_lesson_per_user
    ON attendance_sessions (university_id, user_id, lesson_id)
    WHERE mode = 'lesson' AND lesson_id IS NOT NULL;

CREATE INDEX idx_sessions_lesson
    ON attendance_sessions (university_id, lesson_id)
    WHERE lesson_id IS NOT NULL;

CREATE INDEX idx_sessions_open_closes
    ON attendance_sessions (closes_at)
    WHERE status = 'open' AND mode = 'lesson';

CREATE INDEX idx_sessions_open_day
    ON attendance_sessions (work_date)
    WHERE status = 'open' AND mode = 'day';

ALTER TABLE user_snapshots
    ADD COLUMN IF NOT EXISTS has_student BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE IF NOT EXISTS student_reputations (
    user_id       UUID PRIMARY KEY REFERENCES user_snapshots(user_id) ON DELETE CASCADE,
    university_id TEXT NOT NULL DEFAULT 'kstu',
    score         INT NOT NULL DEFAULT 100 CHECK (score >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

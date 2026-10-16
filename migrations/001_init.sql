-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE user_snapshots (
    user_id           UUID PRIMARY KEY,
    university_id     TEXT NOT NULL DEFAULT 'kstu',
    username          TEXT NOT NULL DEFAULT '',
    first_name        TEXT NOT NULL DEFAULT '',
    last_name         TEXT NOT NULL DEFAULT '',
    middle_name       TEXT NOT NULL DEFAULT '',
    has_student       BOOLEAN NOT NULL DEFAULT FALSE,
    has_employee      BOOLEAN NOT NULL DEFAULT FALSE,
    is_department_head BOOLEAN NOT NULL DEFAULT FALSE,
    department_id     TEXT,
    department_name   TEXT NOT NULL DEFAULT '',
    synced_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE student_reputations (
    user_id       UUID PRIMARY KEY REFERENCES user_snapshots(user_id) ON DELETE CASCADE,
    university_id TEXT NOT NULL DEFAULT 'kstu',
    score         INT NOT NULL DEFAULT 100 CHECK (score >= 0),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- mode: day | lesson
-- acting_as: staff | teacher | student
-- status: open | closed
-- trust: scanned | verified | false_scan (lesson/student primarily)
-- close_reason: manual | auto_cutoff | auto_lesson | NULL when open
CREATE TABLE attendance_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    university_id   TEXT NOT NULL DEFAULT 'kstu',
    user_id         UUID NOT NULL REFERENCES user_snapshots(user_id),
    acting_as       TEXT NOT NULL CHECK (acting_as IN ('staff', 'teacher', 'student')),
    mode            TEXT NOT NULL CHECK (mode IN ('day', 'lesson')),
    work_date       DATE NOT NULL,
    department_id   TEXT,
    room_id         TEXT,
    lesson_id       TEXT,
    opened_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at       TIMESTAMPTZ,
    closes_at       TIMESTAMPTZ,
    status          TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    trust           TEXT NOT NULL DEFAULT 'scanned' CHECK (trust IN ('scanned', 'verified', 'false_scan')),
    close_reason    TEXT CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff', 'auto_lesson')),
    duration_seconds INT,
    display_name    TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_day_open_per_user
    ON attendance_sessions (university_id, user_id, work_date)
    WHERE mode = 'day' AND status = 'open';

CREATE UNIQUE INDEX uq_day_session_per_user
    ON attendance_sessions (university_id, user_id, work_date)
    WHERE mode = 'day';

CREATE UNIQUE INDEX uq_lesson_per_user
    ON attendance_sessions (university_id, user_id, lesson_id)
    WHERE mode = 'lesson' AND lesson_id IS NOT NULL;

CREATE INDEX idx_sessions_user_date ON attendance_sessions (university_id, user_id, work_date);
CREATE INDEX idx_sessions_dept_date ON attendance_sessions (university_id, department_id, work_date);
CREATE INDEX idx_sessions_lesson ON attendance_sessions (university_id, lesson_id) WHERE lesson_id IS NOT NULL;
CREATE INDEX idx_sessions_open_closes ON attendance_sessions (closes_at) WHERE status = 'open' AND mode = 'lesson';
CREATE INDEX idx_sessions_open_day ON attendance_sessions (work_date) WHERE status = 'open' AND mode = 'day';

-- absence / timesheet day codes
CREATE TABLE day_exceptions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    university_id   TEXT NOT NULL DEFAULT 'kstu',
    user_id         UUID NOT NULL REFERENCES user_snapshots(user_id),
    department_id   TEXT,
    work_date       DATE NOT NULL,
    code            TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'approved'
        CHECK (status IN ('pending', 'approved', 'rejected')),
    source          TEXT NOT NULL DEFAULT 'head'
        CHECK (source IN ('employee_request', 'head', 'system')),
    note            TEXT NOT NULL DEFAULT '',
    requested_by    UUID,
    decided_by      UUID,
    decided_at      TIMESTAMPTZ,
    display_name    TEXT NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_exception_approved_day
    ON day_exceptions (university_id, user_id, work_date)
    WHERE status = 'approved';

CREATE INDEX idx_exceptions_dept_date ON day_exceptions (university_id, department_id, work_date);
CREATE INDEX idx_exceptions_pending ON day_exceptions (university_id, department_id, status)
    WHERE status = 'pending';

-- archive mirrors (same shape, no FKs to hot for detach flexibility)
CREATE TABLE attendance_sessions_archive (
    LIKE attendance_sessions INCLUDING DEFAULTS INCLUDING CONSTRAINTS
);
CREATE INDEX idx_sessions_arch_user_date ON attendance_sessions_archive (university_id, user_id, work_date);
CREATE INDEX idx_sessions_arch_dept_date ON attendance_sessions_archive (university_id, department_id, work_date);

CREATE TABLE day_exceptions_archive (
    LIKE day_exceptions INCLUDING DEFAULTS INCLUDING CONSTRAINTS
);
CREATE INDEX idx_exceptions_arch_user_date ON day_exceptions_archive (university_id, user_id, work_date);
CREATE INDEX idx_exceptions_arch_dept_date ON day_exceptions_archive (university_id, department_id, work_date);

CREATE TABLE archive_runs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    university_id TEXT NOT NULL DEFAULT 'kstu',
    cutoff_date   DATE NOT NULL,
    sessions_moved INT NOT NULL DEFAULT 0,
    exceptions_moved INT NOT NULL DEFAULT 0,
    ran_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS archive_runs;
DROP TABLE IF EXISTS day_exceptions_archive;
DROP TABLE IF EXISTS attendance_sessions_archive;
DROP TABLE IF EXISTS day_exceptions;
DROP TABLE IF EXISTS attendance_sessions;
DROP TABLE IF EXISTS student_reputations;
DROP TABLE IF EXISTS user_snapshots;

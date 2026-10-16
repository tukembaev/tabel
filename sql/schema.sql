CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE user_snapshots (
    user_id            UUID PRIMARY KEY,
    university_id      TEXT NOT NULL DEFAULT 'kstu',
    username           TEXT NOT NULL DEFAULT '',
    first_name         TEXT NOT NULL DEFAULT '',
    last_name          TEXT NOT NULL DEFAULT '',
    middle_name        TEXT NOT NULL DEFAULT '',
    has_employee       BOOLEAN NOT NULL DEFAULT FALSE,
    is_department_head BOOLEAN NOT NULL DEFAULT FALSE,
    can_edit_timesheet BOOLEAN NOT NULL DEFAULT FALSE,
    department_id      TEXT,
    department_name    TEXT NOT NULL DEFAULT '',
    synced_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One row per employee per work_date: first scan = in, second scan = out.
-- Lunch is deducted automatically (lunch_seconds), not via extra scans.
-- close_reason: manual | auto_cutoff | head | NULL when open
-- source: scan | head | system
CREATE TABLE attendance_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    university_id    TEXT NOT NULL DEFAULT 'kstu',
    user_id          UUID NOT NULL REFERENCES user_snapshots(user_id),
    work_date        DATE NOT NULL,
    department_id    TEXT,
    entry_building   TEXT NOT NULL DEFAULT '',
    entry_room       TEXT NOT NULL DEFAULT '',
    exit_building    TEXT,
    exit_room        TEXT,
    opened_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at        TIMESTAMPTZ,
    status           TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    close_reason     TEXT CHECK (close_reason IS NULL OR close_reason IN ('manual', 'auto_cutoff', 'head')),
    duration_seconds INT,
    lunch_seconds    INT NOT NULL DEFAULT 0,
    source           TEXT NOT NULL DEFAULT 'scan' CHECK (source IN ('scan', 'head', 'system')),
    edited_by        UUID,
    edited_at        TIMESTAMPTZ,
    note             TEXT NOT NULL DEFAULT '',
    display_name     TEXT NOT NULL DEFAULT '',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_day_session_per_user
    ON attendance_sessions (university_id, user_id, work_date);

CREATE UNIQUE INDEX uq_day_open_per_user
    ON attendance_sessions (university_id, user_id, work_date)
    WHERE status = 'open';

CREATE INDEX idx_sessions_user_date ON attendance_sessions (university_id, user_id, work_date);
CREATE INDEX idx_sessions_dept_date ON attendance_sessions (university_id, department_id, work_date);
CREATE INDEX idx_sessions_open_day ON attendance_sessions (work_date) WHERE status = 'open';

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

CREATE TABLE timesheet_edits (
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

CREATE INDEX idx_timesheet_edits_dept ON timesheet_edits (university_id, department_id, created_at DESC);
CREATE INDEX idx_timesheet_edits_user ON timesheet_edits (university_id, user_id, work_date);

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
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    university_id    TEXT NOT NULL DEFAULT 'kstu',
    cutoff_date      DATE NOT NULL,
    sessions_moved   INT NOT NULL DEFAULT 0,
    exceptions_moved INT NOT NULL DEFAULT 0,
    ran_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

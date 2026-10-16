-- name: CreateDaySession :one
INSERT INTO attendance_sessions (
    university_id, user_id, work_date, department_id,
    entry_building, entry_room, opened_at, status, source, display_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, 'open', 'scan', $8
)
RETURNING *;

-- name: GetLatestDaySession :one
SELECT * FROM attendance_sessions
WHERE university_id = $1 AND user_id = $2 AND work_date = $3
ORDER BY opened_at DESC
LIMIT 1;

-- name: HasDaySession :one
SELECT EXISTS (
    SELECT 1 FROM attendance_sessions
    WHERE university_id = $1 AND user_id = $2 AND work_date = $3
);

-- name: CloseDaySession :one
UPDATE attendance_sessions
SET status = 'closed',
    closed_at = $2,
    close_reason = $3,
    exit_building = COALESCE($4, exit_building),
    exit_room = COALESCE($5, exit_room),
    duration_seconds = $6,
    lunch_seconds = $7,
    updated_at = NOW()
WHERE id = $1 AND status = 'open'
RETURNING *;

-- name: OverrideAutoClose :one
UPDATE attendance_sessions
SET status = 'closed',
    closed_at = $2,
    close_reason = 'manual',
    exit_building = COALESCE($3, exit_building),
    exit_room = COALESCE($4, exit_room),
    duration_seconds = $5,
    lunch_seconds = $6,
    updated_at = NOW()
WHERE id = $1 AND status = 'closed' AND close_reason = 'auto_cutoff'
RETURNING *;

-- name: ListOpenDaySessionsForDate :many
SELECT * FROM attendance_sessions
WHERE university_id = $1 AND status = 'open' AND work_date = $2;

-- name: GetSessionByID :one
SELECT * FROM attendance_sessions WHERE id = $1;

-- name: DeleteSessionsForUserDate :exec
DELETE FROM attendance_sessions
WHERE university_id = $1 AND user_id = $2 AND work_date = $3;

-- name: UpsertHeadDaySession :one
INSERT INTO attendance_sessions (
    university_id, user_id, work_date, department_id,
    entry_building, entry_room, opened_at, closed_at, status,
    close_reason, duration_seconds, lunch_seconds, display_name,
    source, edited_by, edited_at, note
) VALUES (
    $1, $2, $3, $4, '', '', $5, $6, 'closed',
    'head', $7, $8, $9, 'head', $10, NOW(), $11
)
ON CONFLICT (university_id, user_id, work_date) DO UPDATE SET
    opened_at = EXCLUDED.opened_at,
    closed_at = EXCLUDED.closed_at,
    status = 'closed',
    close_reason = 'head',
    duration_seconds = EXCLUDED.duration_seconds,
    lunch_seconds = EXCLUDED.lunch_seconds,
    source = 'head',
    edited_by = EXCLUDED.edited_by,
    edited_at = NOW(),
    note = EXCLUDED.note,
    display_name = EXCLUDED.display_name,
    updated_at = NOW()
RETURNING *;

-- name: ListUserSessionsInRange :many
SELECT * FROM attendance_sessions
WHERE university_id = $1 AND user_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date, opened_at;

-- name: ListDepartmentSessionsInRange :many
SELECT * FROM attendance_sessions
WHERE university_id = $1 AND department_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date, display_name, opened_at;

-- name: ListUserSessionsArchiveInRange :many
SELECT * FROM attendance_sessions_archive
WHERE university_id = $1 AND user_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date, opened_at;

-- name: ListDepartmentSessionsArchiveInRange :many
SELECT * FROM attendance_sessions_archive
WHERE university_id = $1 AND department_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date, display_name, opened_at;

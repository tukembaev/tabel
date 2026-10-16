-- name: MoveSessionToArchive :exec
INSERT INTO attendance_sessions_archive
SELECT s.* FROM attendance_sessions s WHERE s.id = $1;

-- name: DeleteSession :exec
DELETE FROM attendance_sessions WHERE id = $1;

-- name: ListSessionIDsBefore :many
SELECT s.id FROM attendance_sessions s
WHERE s.university_id = $1 AND s.work_date < $2
LIMIT $3;

-- name: MoveExceptionToArchive :exec
INSERT INTO day_exceptions_archive
SELECT e.* FROM day_exceptions e WHERE e.id = $1;

-- name: DeleteException :exec
DELETE FROM day_exceptions WHERE id = $1;

-- name: ListExceptionIDsBefore :many
SELECT e.id FROM day_exceptions e
WHERE e.university_id = $1 AND e.work_date < $2
LIMIT $3;

-- name: InsertArchiveRun :one
INSERT INTO archive_runs (university_id, cutoff_date, sessions_moved, exceptions_moved)
VALUES ($1, $2, $3, $4)
RETURNING *;

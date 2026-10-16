-- name: CreateDayException :one
INSERT INTO day_exceptions (
    university_id, user_id, department_id, work_date, code, status, source,
    note, requested_by, decided_by, decided_at, display_name
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
RETURNING *;

-- name: GetApprovedException :one
SELECT * FROM day_exceptions
WHERE university_id = $1 AND user_id = $2 AND work_date = $3 AND status = 'approved';

-- name: GetDayExceptionByID :one
SELECT * FROM day_exceptions WHERE id = $1;

-- name: ListPendingExceptionsByDept :many
SELECT * FROM day_exceptions
WHERE university_id = $1 AND department_id = $2 AND status = 'pending'
ORDER BY work_date DESC, created_at DESC;

-- name: ListUserExceptionsInRange :many
SELECT * FROM day_exceptions
WHERE university_id = $1 AND user_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date;

-- name: ListDepartmentExceptionsInRange :many
SELECT * FROM day_exceptions
WHERE university_id = $1 AND department_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date, display_name;

-- name: ListUserExceptionsArchiveInRange :many
SELECT * FROM day_exceptions_archive
WHERE university_id = $1 AND user_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date;

-- name: ListDepartmentExceptionsArchiveInRange :many
SELECT * FROM day_exceptions_archive
WHERE university_id = $1 AND department_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY work_date, display_name;

-- name: DecideException :one
UPDATE day_exceptions
SET status = $2,
    decided_by = $3,
    decided_at = NOW(),
    updated_at = NOW()
WHERE id = $1 AND status = 'pending'
RETURNING *;

-- name: DeleteExceptionsForUserDate :exec
DELETE FROM day_exceptions
WHERE university_id = $1 AND user_id = $2 AND work_date = $3;


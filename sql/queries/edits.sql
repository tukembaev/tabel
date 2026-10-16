-- name: InsertTimesheetEdit :one
INSERT INTO timesheet_edits (
    university_id, department_id, editor_id, user_id, work_date, action, note, payload
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: ListTimesheetEditsByDept :many
SELECT * FROM timesheet_edits
WHERE university_id = $1 AND department_id = $2 AND work_date >= $3 AND work_date <= $4
ORDER BY created_at DESC;

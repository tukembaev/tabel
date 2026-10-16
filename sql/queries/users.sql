-- name: UpsertUserSnapshot :one
INSERT INTO user_snapshots (
    user_id, university_id, username, first_name, last_name, middle_name,
    has_employee, is_department_head, can_edit_timesheet, department_id, department_name, synced_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW()
)
ON CONFLICT (user_id) DO UPDATE SET
    university_id = EXCLUDED.university_id,
    username = EXCLUDED.username,
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    middle_name = EXCLUDED.middle_name,
    has_employee = EXCLUDED.has_employee,
    is_department_head = EXCLUDED.is_department_head,
    can_edit_timesheet = EXCLUDED.can_edit_timesheet,
    department_id = EXCLUDED.department_id,
    department_name = EXCLUDED.department_name,
    synced_at = NOW(),
    updated_at = NOW()
RETURNING *;

-- name: GetUserSnapshot :one
SELECT * FROM user_snapshots WHERE user_id = $1;

-- name: ListEmployeesByDepartment :many
SELECT * FROM user_snapshots
WHERE university_id = $1 AND department_id = $2 AND has_employee = TRUE
ORDER BY last_name, first_name, username;

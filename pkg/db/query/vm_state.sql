-- name: UpsertVMState :exec
INSERT INTO vm_state (id, phase, pid, boot_time, seen_at)
VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    phase = excluded.phase,
    pid = excluded.pid,
    boot_time = excluded.boot_time,
    seen_at = excluded.seen_at;

-- name: GetVMState :one
SELECT id, phase, pid, boot_time, seen_at FROM vm_state WHERE id = ?;

-- name: ListVMStates :many
SELECT id, phase, pid, boot_time, seen_at FROM vm_state;

-- name: DeleteVMState :exec
DELETE FROM vm_state WHERE id = ?;

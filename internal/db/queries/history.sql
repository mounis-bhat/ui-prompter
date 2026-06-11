-- name: ListHistory :many
SELECT * FROM history ORDER BY created_at DESC;

-- name: AddHistory :one
INSERT INTO history (source_type, source_uri, prompt)
VALUES (?, ?, ?)
RETURNING *;

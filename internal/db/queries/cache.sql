-- name: GetCache :one
SELECT response FROM cache WHERE hash = ?;

-- name: SetCache :exec
INSERT INTO cache (hash, response) VALUES (?, ?)
ON CONFLICT(hash) DO UPDATE SET response = excluded.response;

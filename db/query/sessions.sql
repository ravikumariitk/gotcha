-- name: CreateSession :one
INSERT INTO sessions (id, user_id, expires_at) VALUES ($1, $2, $3) RETURNING *;
-- name: GetSessionUser :one
SELECT u.* FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.id = $1 AND s.expires_at > now();
-- name: DeleteSession :exec
DELETE FROM sessions WHERE id = $1;

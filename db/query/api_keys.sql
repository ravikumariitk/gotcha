-- name: CreateAPIKey :one
INSERT INTO api_keys (project_id, key_prefix, key_hash, name) VALUES ($1, $2, $3, $4) RETURNING *;
-- name: GetActiveAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = $1 AND revoked_at IS NULL;
-- name: RevokeAPIKey :exec
UPDATE api_keys SET revoked_at = now() WHERE id = $1 AND project_id = $2;
-- name: ListAPIKeysForProject :many
SELECT * FROM api_keys WHERE project_id = $1 ORDER BY created_at DESC;

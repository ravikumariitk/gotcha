-- name: CreateUser :one
INSERT INTO users (email, name, password_hash) VALUES ($1, $2, $3) RETURNING *;
-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;
-- name: CreateProject :one
INSERT INTO projects (owner_id, name) VALUES ($1, $2) RETURNING *;
-- name: ListProjectsForUser :many
SELECT * FROM projects WHERE owner_id = $1 ORDER BY created_at DESC;

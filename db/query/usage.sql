-- name: CreateVerificationEvent :exec
INSERT INTO verification_events (project_id, outcome) VALUES ($1, $2);
-- name: GetProjectUsage :one
SELECT COUNT(*) FILTER (WHERE outcome = 'challenge') AS total_challenges,
       COUNT(*) FILTER (WHERE outcome = 'passed') AS passed,
       COUNT(*) FILTER (WHERE outcome = 'blocked') AS blocked
FROM verification_events WHERE project_id = $1 AND created_at >= $2;

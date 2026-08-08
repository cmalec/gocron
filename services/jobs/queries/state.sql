-- name: GetJobStates :many
SELECT
  job_slug,
  disabled
FROM
  job_state;

-- name: SetJobState :exec
INSERT INTO
  job_state (job_slug, disabled)
VALUES
  (?, ?) ON CONFLICT (job_slug) DO
UPDATE
SET
  disabled = excluded.disabled;

-- name: DeleteObsoleteJobStates :exec
DELETE FROM job_state
WHERE
  job_slug NOT IN (sqlc.slice (job_slugs));

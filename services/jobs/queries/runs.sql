-- name: GetRuns :many
SELECT
  id,
  job_name,
  job_slug,
  status_id,
  start_time,
  end_time
FROM
  (
    SELECT
      id,
      job_name,
      job_slug,
      status_id,
      start_time,
      end_time
    FROM
      runs
    WHERE
      job_slug = ?
    ORDER BY
      start_time DESC
    LIMIT
      ?
  ) AS sub
ORDER BY
  start_time ASC;

-- name: GetThreeRunsPerJobName :many
WITH
  ranked_runs AS (
    SELECT
      id,
      job_name,
      job_slug,
      status_id,
      start_time,
      end_time,
      ROW_NUMBER() OVER (
        PARTITION BY
          job_slug
        ORDER BY
          start_time DESC
      ) AS rn
    FROM
      runs
  )
SELECT
  id,
  job_name,
  job_slug,
  status_id,
  start_time,
  end_time
FROM
  ranked_runs
WHERE
  rn <= 3
ORDER BY
  job_slug,
  rn DESC;

-- name: CreateRun :one
INSERT INTO
  runs (job_name, job_slug, status_id, start_time)
VALUES
  (?, ?, ?, ?) RETURNING *;

-- name: UpdateRun :one
UPDATE runs
SET
  status_id = ?,
  end_time = ?
WHERE
  id = ? RETURNING *;

-- name: IsIdle :one
SELECT
  CAST(
    NOT EXISTS (
      SELECT
        1
      FROM
        runs
      WHERE
        status_id = 1
    ) AS INTEGER
  ) AS is_idle;

-- name: DeleteOldRuns :exec
DELETE FROM runs
WHERE
  start_time < ?;

-- name: DeleteObsoleteRuns :exec
DELETE FROM runs
WHERE
  job_slug NOT IN (sqlc.slice (job_slugs));

-- name: StopRunning :exec
UPDATE runs
SET
  status_id = 4,
  end_time = STRFTIME ('%s', 'now') * 1000
WHERE
  status_id = 1
  AND end_time IS NULL;

-- name: GetDailyRunStats :many
SELECT
  CAST(
    STRFTIME (
      '%Y-%m-%d',
      start_time / 1000,
      'unixepoch',
      'localtime'
    ) AS TEXT
  ) AS day,
  CAST(
    SUM(
      CASE
        WHEN status_id = 3 THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS succeeded,
  CAST(
    SUM(
      CASE
        WHEN status_id IN (2, 4) THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS failed,
  COUNT(*) AS total
FROM
  runs
WHERE
  job_slug = ?
  AND start_time >= ?
GROUP BY
  day
ORDER BY
  day;

-- name: GetRecentRuns :many
SELECT
  id,
  job_name,
  job_slug,
  status_id,
  start_time,
  end_time
FROM
  runs
ORDER BY
  start_time DESC
LIMIT
  ?;

-- name: GetJobSuccessRates :many
SELECT
  job_slug,
  CAST(
    SUM(
      CASE
        WHEN status_id = 3 THEN 1
        ELSE 0
      END
    ) AS INTEGER
  ) AS succeeded,
  COUNT(*) AS total
FROM
  runs
WHERE
  status_id != 1
GROUP BY
  job_slug;

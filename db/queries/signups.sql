-- name: HasName :one
SELECT EXISTS(
    SELECT 1 FROM telive.signups
    WHERE LOWER(name) = LOWER($1)
      AND created_at >= CURRENT_DATE
) AS exists;
-- name: HasName :one
SELECT EXISTS(
    SELECT 1 FROM telive.signups
    WHERE LOWER(name) = LOWER($1)
      AND created_at >= CURRENT_DATE
) AS exists;

-- name: ListTodayPositions :many
SELECT id, position FROM telive.signups
WHERE created_at >= CURRENT_DATE
ORDER BY position ASC;

-- name: ListTodayEntries :many
SELECT qe.id, qe.name, s.id AS song_id, s.title, s.artist, s.tab_url, es.performed, qe.times_on_stage
FROM telive.signups qe
JOIN telive.entry_songs es ON es.entry_id = qe.id
JOIN telive.songs s ON s.id = es.song_id
WHERE qe.created_at >= CURRENT_DATE
ORDER BY qe.position ASC, es.sort_order ASC;
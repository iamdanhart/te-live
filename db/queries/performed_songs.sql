-- name: ListPerformedToday :many
SELECT ps.singer, s.id, s.title, s.artist, COALESCE(s.tab_url, '') AS tab_url
FROM telive.performed_songs ps
JOIN telive.songs s ON s.id = ps.song_id
WHERE ps.performed_at >= CURRENT_DATE
ORDER BY ps.performed_at ASC;
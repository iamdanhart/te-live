-- name: ListSongs :many
SELECT id, title, artist, COALESCE(tab_url, '') AS tab_url
FROM telive.songs
ORDER BY title ASC;
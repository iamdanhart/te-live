-- name: ListSongs :many
SELECT id, title, artist, tab_url
FROM telive.songs
ORDER BY title ASC;
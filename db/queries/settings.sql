-- name: GetSignupsOpen :one
SELECT value FROM telive.settings WHERE key = 'signups_open';
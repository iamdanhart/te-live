-- name: GetSignupsOpen :one
SELECT value FROM telive.settings WHERE key = 'signups_open';

-- name: ToggleSignupsOpen :one
UPDATE telive.settings
SET value = CASE WHEN value = 'true' THEN 'false' ELSE 'true' END
WHERE key = 'signups_open'
RETURNING value;
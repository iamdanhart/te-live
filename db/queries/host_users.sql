-- name: ListActivePasscodeHashes :many
SELECT passcode_hash FROM telive.host_users WHERE active = TRUE;
CREATE SCHEMA IF NOT EXISTS telive;

CREATE TABLE telive.songs (
    id      SERIAL PRIMARY KEY,
    title   TEXT NOT NULL,
    artist  TEXT NOT NULL,
    tab_url TEXT
);

CREATE TABLE telive.signups (
    id             SERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    position       FLOAT8 NOT NULL,
    times_on_stage INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE telive.entry_songs (
    id         SERIAL PRIMARY KEY,
    entry_id   INT NOT NULL REFERENCES telive.signups(id) ON DELETE CASCADE,
    song_id    INT NOT NULL REFERENCES telive.songs(id),
    performed  BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT NOT NULL DEFAULT 0
);

CREATE TABLE telive.performed_songs (
    id           SERIAL PRIMARY KEY,
    singer       TEXT NOT NULL,
    song_id      INT NOT NULL REFERENCES telive.songs(id),
    performed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE telive.settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE telive.host_users (
    id            SERIAL PRIMARY KEY,
    label         TEXT NOT NULL,
    passcode_hash TEXT NOT NULL,
    active        BOOLEAN NOT NULL DEFAULT TRUE
);
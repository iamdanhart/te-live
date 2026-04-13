--liquibase formatted sql

--changeset dan:1
CREATE TABLE songs (
    id      SERIAL PRIMARY KEY,
    title   TEXT NOT NULL,
    artist  TEXT NOT NULL,
    tab_url TEXT
);

--changeset dan:2
CREATE TABLE signups (
    id             SERIAL PRIMARY KEY,
    name           TEXT NOT NULL,
    position       FLOAT8 NOT NULL,
    times_on_stage INT  NOT NULL DEFAULT 0,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

--changeset dan:2b
CREATE UNIQUE INDEX signups_position_date_idx ON signups (position, (created_at::date));

--changeset dan:3
CREATE TABLE entry_songs (
    id         SERIAL PRIMARY KEY,
    entry_id   INT     NOT NULL REFERENCES signups(id) ON DELETE CASCADE,
    song_id    INT     NOT NULL REFERENCES songs(id),
    performed  BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order INT     NOT NULL DEFAULT 0
);

--changeset dan:4
CREATE TABLE performed_songs (
    id           SERIAL PRIMARY KEY,
    singer       TEXT NOT NULL,
    song_id      INT  NOT NULL REFERENCES songs(id),
    performed_at TIMESTAMP NOT NULL DEFAULT NOW()
);

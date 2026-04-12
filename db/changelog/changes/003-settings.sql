--liquibase formatted sql

--changeset dan:6
CREATE TABLE settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

--changeset dan:7
INSERT INTO settings (key, value) VALUES ('signups_open', 'true');
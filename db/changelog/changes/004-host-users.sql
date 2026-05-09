--liquibase formatted sql

--changeset dan:8
CREATE TABLE host_users (
    id            SERIAL PRIMARY KEY,
    label         TEXT    NOT NULL,
    passcode_hash TEXT    NOT NULL,
    active        BOOLEAN NOT NULL DEFAULT TRUE
);
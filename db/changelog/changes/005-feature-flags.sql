--liquibase formatted sql

--changeset dan:9
CREATE TABLE feature_flags (
    key     TEXT    PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT FALSE
);
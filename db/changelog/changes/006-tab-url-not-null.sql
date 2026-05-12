--liquibase formatted sql

--changeset dan:10
ALTER TABLE telive.songs ALTER COLUMN tab_url SET NOT NULL;
ALTER TABLE telive.songs ADD CONSTRAINT songs_tab_url_https CHECK (tab_url LIKE 'https://%');
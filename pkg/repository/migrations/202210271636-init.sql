-- noinspection SqlNoDataSourceInspectionForFile

-- +migrate Up
CREATE TABLE IF NOT EXISTS users
(
    id         bigserial     PRIMARY KEY,
    username   varchar       UNIQUE NOT NULL,
    password    text          NOT NULL
);

-- +migrate Down
DROP TABLE users CASCADE;
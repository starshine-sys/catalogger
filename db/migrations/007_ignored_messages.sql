-- +migrate Up

-- 2022-07-13: Add ignored messages table

create table ignored_messages (
    id  bigint  primary key
);

-- +migrate Up

-- 2021-12-19: Change guilds table banned_systems column to text array

alter table guilds alter column banned_systems set data type text[];

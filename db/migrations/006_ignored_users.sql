-- +migrate Up

-- 2022-03-20: Add ignored users column

alter table guilds add column ignored_users bigint[] not null default array[]::bigint[];

-- +migrate Up

alter table guilds add column key_roles bigint[] not null default array[]::bigint[];

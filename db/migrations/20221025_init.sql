-- +migrate Up

-- 2022-10-25: Initial database schema
-- Again consolidates every old migration into a single schema.
create table if not exists guilds (
    id  bigint  primary key,

    channels    json    not null,
    redirects   json    not null default '{}',

    ignored_channels    bigint[]    not null    default array[]::bigint[],
    banned_systems      text[]      not null    default array[]::text[],
    key_roles           bigint[]    not null    default array[]::bigint[],
    ignored_users       bigint[]    not null    default array[]::bigint[]
);

create table if not exists messages (
    msg_id      bigint  primary key,
    user_id     bigint  not null    default 0,
    channel_id  bigint  not null    default 0,
    server_id   bigint  not null    default 0,

    username    text not null default '',
    member      text,
    system      text,

    content     text    not null,
    metadata    bytea
);

create table if not exists ignored_messages (
    id  bigint  primary key
);

create table if not exists invites (
    guild_id    bigint  not null,
    code        text    primary key,
    name        text    not null
);

create table if not exists watchlist (
    guild_id    bigint,
    user_id     bigint,

    moderator   bigint not null,
    added       timestamp   not null    default (current_timestamp at time zone 'utc'),

    reason  text    not null,

    primary key (guild_id, user_id)
);

create index if not exists messages_server_id_idx on messages (server_id);
create index if not exists invites_guild_idx on invites (guild_id);
create index if not exists watchlist_guild_id_idx on watchlist (guild_id);

-- remove old migration history
drop table if exists gorp_migrations;

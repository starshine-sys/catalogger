-- +migrate Up

-- 2022-10-25: Initial database schema
-- Again consolidates every old migration into a single schema.
create table guilds (
    id  bigint  primary key,

    channels  jsonb not null,
    redirects jsonb not null default '{}',
    ignores   jsonb not null default '{}',

    banned_systems text[]   not null default array[]::text[],
    key_roles      bigint[] not null default array[]::bigint[]
);

create table messages (
    id          bigint  primary key,
    user_id     bigint  not null,
    channel_id  bigint  not null,
    guild_id    bigint  not null,

    username    bytea not null,
    member      text,
    system      text,

    content     bytea not null,
    metadata    bytea,

    -- statistics to see if we can also log attachments in the future
    attachment_size integer not null default 0
);

create table ignored_messages (
    id  bigint  primary key
);

create table invites (
    guild_id    bigint  not null,
    code        text    primary key,
    name        text    not null
);

create table watchlist (
    guild_id    bigint,
    user_id     bigint,

    moderator   bigint not null,
    added       timestamp   not null    default (current_timestamp at time zone 'utc'),

    reason  text    not null,

    primary key (guild_id, user_id)
);

create index messages_guild_id_idx on messages (guild_id);
create index invites_guild_idx on invites (guild_id);
create index watchlist_guild_id_idx on watchlist (guild_id);

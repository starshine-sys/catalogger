-- +migrate Up

-- 2021-07-14: Initial database schema
-- Everything here should assume either an empty database or a fully up-to-date database from the migration files using `tern` (in migrations_legacy/ in the root directory)
create table if not exists guilds (
    id  bigint  primary key,

    channels    json    not null,
    redirects   json    not null default '{}',

    ignored_channels    bigint[]    not null    default array[]::bigint[],
    banned_systems      char(5)[]   not null    default array[]::char(5)[]
);

create table if not exists pk_messages (
    msg_id      bigint  primary key,
    user_id     bigint  not null    default 0,
    channel_id  bigint  not null    default 0,
    server_id   bigint  not null    default 0,

    username    text    not null,
    member      text    not null,
    system      text    not null,

    content     text    not null
);

create table if not exists messages (
    msg_id  bigint  primary key,
    user_id     bigint  not null    default 0,
    channel_id  bigint  not null    default 0,
    server_id   bigint  not null    default 0,

    content     text    not null
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
create index if not exists pk_messages_server_id_idx on pk_messages (server_id);
create index if not exists invites_guild_idx on invites (guild_id);
create index if not exists watchlist_guild_id_idx on watchlist (guild_id);

create or replace view server_messages as select distinct messages.server_id, (
    (select count(msg_id) from messages as m where m.server_id = messages.server_id) +
    (select count(msg_id) from pk_messages where server_id = messages.server_id)
    ) as total,
    (select count(msg_id) from messages as m where m.server_id = messages.server_id) as normal,
    (select count(msg_id) from pk_messages where server_id = messages.server_id) as proxied
from messages group by server_id order by total desc;

create table guilds (
    id  bigint  primary key,

    channels    json    not null,

    ignored_channels    bigint[]    not null    default array[]::bigint[]
);

create table pk_messages (
    msg_id      bigint  primary key,
    user_id     bigint  not null    default 0,
    channel_id  bigint  not null    default 0,
    server_id   bigint  not null    default 0,

    username    text    not null,
    member      text    not null,
    system      text    not null,

    content     text    not null
);

create table messages (
    msg_id  bigint  primary key,
    user_id     bigint  not null    default 0,
    channel_id  bigint  not null    default 0,
    server_id   bigint  not null    default 0,

    content     text    not null
);

---- create above / drop below ----

drop table guilds;
drop table pk_messages;
drop table messages;
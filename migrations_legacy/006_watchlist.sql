create table watchlist (
    guild_id    bigint,
    user_id     bigint,

    moderator   bigint not null,
    added       timestamp   not null    default (current_timestamp at time zone 'utc'),

    reason  text    not null,

    primary key (guild_id, user_id)
);

create index watchlist_guild_id_idx on watchlist (guild_id);

---- create above / drop below ----

drop table watchlist;
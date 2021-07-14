create table invites (
    guild_id    bigint  not null,
    code        text    primary key,
    name        text    not null
);

create index invites_guild_idx on invites (guild_id);

---- create above / drop below ----

drop table invites;
alter table guilds add column banned_systems char(5)[] not null default array[]::char(5)[];

---- create above / drop below ----

alter table guilds drop column banned_systems;
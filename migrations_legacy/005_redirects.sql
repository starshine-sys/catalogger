alter table guilds add column redirects json not null default '{}';

---- create above / drop below ----

alter table guilds drop column redirects;
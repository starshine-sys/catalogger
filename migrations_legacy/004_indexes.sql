create index messages_server_id_idx on messages (server_id);
create index pk_messages_server_id_idx on pk_messages (server_id);

create view server_messages as select distinct messages.server_id, (
    (select count(msg_id) from messages as m where m.server_id = messages.server_id) +
    (select count(msg_id) from pk_messages where server_id = messages.server_id)
    ) as total,
    (select count(msg_id) from messages as m where m.server_id = messages.server_id) as normal,
    (select count(msg_id) from pk_messages where server_id = messages.server_id) as proxied
from messages group by server_id order by total desc;

---- create above / drop below ----

drop index messages_server_id_idx;
drop index pk_messages_server_id_idx;

drop view server_messages;
-- +migrate Up

-- 2021-08-22: Merge `pk_messages` into `messages`
-- The pk_messages table is not dropped.
-- You should do this manually once the migration completes successfully :]

-- this should always be filled, the default is for existing messages
alter table messages add column username text not null default '';

-- both nullable
alter table messages add column member text;
alter table messages add column system text;

-- copy messages
-- remove messages that already exist in the `pk_messages` table
delete from messages where msg_id = any(select msg_id from pk_messages);
-- actually copy the messages
insert into messages (msg_id, user_id, channel_id, server_id, username, member, system, content)
select msg_id, user_id, channel_id, server_id, username, member, system, content from pk_messages;
-- +migrate Up

-- 2021-10-27: add metadata column for messages
-- metadata is encrypted JSON data about the message--author (for webhook messages), embeds, attachments

alter table messages add column metadata bytea;

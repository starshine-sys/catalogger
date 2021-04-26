# Bot usage

To invite the bot to your server, use [this link](https://discord.com/api/oauth2/authorize?client_id=830819903371739166&permissions=537259248&scope=bot%20applications.commands).

Catalogger's prefix is `cl!` or `lg!`. Self-hosted instances can use any prefix.

A list of all commands can be shown with `cl!help commands`.

## Getting started

To start logging events, use the command `cl!setchannel` with a *comma-separated* list of [events](#events).
- For example: `cl!setchannel message_update,message_delete`

To *disable* logging for an event, use `cl!setchannel` with the `--clear` flag.
- For example, `cl!setchannel --clear message_update` will stop logging the message update event.

To ignore events from a channel, use `cl!ignorechannel` in that channel.  
To stop ignoring the channel, run the command again.

### Logging invites

Invites are logged by default (although the bot needs both "Manage Server" and "Manage Channels" for it to work).

You can give invites names, to track where people are coming from.

To do this, use the `cl!invites name` command with the invite code and the name you want to give to the invite.

You can use `cl!invites` to list all of the server's invites.

## Troubleshooting

If the bot stops logging an event, try the `cl!permcheck` command;
if that's all clear, but the bot still isn't logging, try the `cl!clearcache` command.

If it's still not working, feel free to join the [support server](https://discord.gg/anzCcFKBk4) and ask there!

## Resetting your data

To delete *all* of your server's data, use the `cl!cleardata` command.
This will reset your server's configuration, and delete all cached messages.
**This process is irreversible.**

## Events

The following events are implemented:

- `MESSAGE_DELETE`: deleted messages, both normal and PluralKit messages
- `MESSAGE_DELETE_BULK`: bulk message deletions
- `MESSAGE_UPDATE`: edited messages
- `GUILD_MEMBER_ADD`: new member joining
- `GUILD_MEMBER_REMOVE`: member leaving
- `INVITE_CREATE`: created invites
- `INVITE_DELETE`: deleted invites
- `GUILD_BAN_ADD`: banned users
- `GUILD_BAN_REMOVE`: unbanned users
- `GUILD_MEMBER_UPDATE`: role updates
- `GUILD_MEMBER_NICK_UPDATE`: username/nickname updates
- `CHANNEL_CREATE`: channel creations
- `CHANNEL_UPDATE`: channel updates
- `CHANNEL_DELETE`: channel deletions
- `GUILD_ROLE_CREATE`: role creations
- `GUILD_ROLE_UPDATE`: role updates
- `GUILD_ROLE_DELETE`: role deletions

The following events are not yet implemented:

- `GUILD_UPDATE`: server updates, such as name and icon changes
- `GUILD_EMOJIS_UPDATE`: changes to a server's custom emotes
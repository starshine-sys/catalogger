# Bot usage

To invite the bot to your server, use [this link](https://discord.com/api/oauth2/authorize?client_id=830819903371739166&permissions=537259248&scope=bot%20applications.commands).

Catalogger's prefix is `lg!`. Self-hosted instances can use any prefix.

To start logging events, use the command `lg!setchannel` with a *comma-separated* list of [events](#events).  
For example: `lg!setchannel message_update,message_delete`  
To *disable* logging for an event, use `lg!setchannel` with the `--clear` flag.  
For example, `lg!setchannel --clear message_update` will stop logging the message update event.

To ignore events from a channel, use `lg!ignorechannel` in that channel.  
To stop ignoring the channel, run the command again.

If the bot stops logging an event, try the `lg!clearcache` command.  
If it's still not working, feel free to join the [support server](https://discord.gg/anzCcFKBk4) and ask there!

To delete *all* of your server's data, use the `lg!cleardata` command.  
This will reset your server's configuration, and delete all cached messages.  
**This process is irreversible.**

## Events

The following events are implemented:

- `MESSAGE_DELETE`: deleted messages, both normal and PluralKit messages
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
- `MESSAGE_DELETE_BULK`: bulk message deletions
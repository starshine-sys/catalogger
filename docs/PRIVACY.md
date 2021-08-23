# Privacy policy

We're not lawyers and never will be, and we don't want to write a document no one can (or wants to) read.  
That being said, this bot handles sensitive information, so here's a list of what the bot collects.

By using Catalogger's commands, or by participating in a server where it's logging events,
you consent to have your data processed by the bot.

This is the data Catalogger collects:

- Messages, in servers where edited and/or deleted message logging is enabled.
- Server-specific settings: which channels to log to, and which channels to ignore.

Messages are stored encrypted in the database, and automatically deleted after fifteen days.  
This retention time may change in the future; however, it will not be extended, only lowered.

This is the data Catalogger fetches from Discord, and is stored while the bot is running:

- User information: IDs, usernames, and avatars.
- Member information: Nicknames and roles.
- Server information: all channels and all roles in a server.
- Webhooks used for logging.

Additionally, the dashboard collects the following data:

- The servers a logged-in user is in, to check their permissions and provide a list of all servers they can manage.

To clear your server's data, use the `lg!cleardata` command.  
Catalogger will stop collecting any information when that command is used, unless you reenable logging afterwards.

Note that that command won't remove any data from database backups. Contact us if you want those wiped too.
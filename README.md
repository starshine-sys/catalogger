# Catalogger

A logger bot that integrates with PluralKit's message proxying.  

For a usage guide, check out docs/USAGE.md

To invite Catalogger to your server, use [this link](https://discord.com/api/oauth2/authorize?client_id=830819903371739166&permissions=537259249&scope=bot%20applications.commands).

## Hosting Catalogger yourself

A logging bot, by its nature, collects a *lot* of potentially sensitive data. All message content is stored encrypted, but even then, you might not be comfortable adding a third-party bot for logging. For that reason, we provide an easy way of self-hosting Catalogger using Docker.

(We're not sure if the Docker image will run on Windows--it's only been tested on Linux. For best results, use a Linux system!)

### Docker

First off, you need Docker and docker-compose installed. 

Copy `.env.example` in this directory to `.env`. For Docker, only the `TOKEN`, `PREFIXES`, `OWNER`, and `AES_KEY` lines are needed, but you can keep the `DATABASE_URL` and `REDIS` keys anyway--they won't affect anything. **Make sure to replace *all* default values!**

Your `.env` file should look something like this:

```
TOKEN=NzMwOTE2MTgxMjQ5ODg0MTgx.YANs0q.amdzZGhmZ2prc2RmaGdqa3NoZmh
PREFIXES=cl!
OWNER=594423578915786351
AES_KEY=ad97zmwmo7nm9TUU9K2XsDozTclX+nJ7+WcS/NVoVAU=
```

Then, just run `docker-compose up` (optionally adding the `-d` flag to run it in the background) to run the bot!

### Dockern't

If for whatever reason you can't or don't want to use Docker, it's still possible to run Catalogger yourself.

#### Requirements

- Go 1.16
- PostgreSQL 12
- Redis 6.2

#### Installation

1. Create a Postgres database
2. Copy `.env.example` to `.env` and fill it in
3. Build the bot with `build.sh`
4. Run the resulting executable

### Configuration

Catalogger is configured through a `.env` file. The following keys are available:

- `TOKEN`: Discord bot token. **Required**
- `DATBASE_URL`: Connection string to a PostgreSQL database. **Required, except for Docker**
- `REDIS`: Connection string to a Redis instance. **Required, except for Docker**
- `OWNER`: Owner user ID, the only account that can use owner-only commands. **Required**
- `AES_KEY`: the key used to encrypt message content. **Required**
  This should be a base64 string, such as from [here](https://generate.plus/en/base64).
- `PREFIXES`: a *comma-separated* list of prefixes for commands. **Required**  
  For example, `cl!,lg!` means both `cl!` and `lg!` are prefixes used to execute commands.
- `COMMANDS_GUILD_ID`: if set to `true`, the *only* server ID that slash commands will be synced to.
  Recommended to set if your instance will only be used in a single server, otherwise, only enable during development.
- `SYNC_COMMANDS`: if not set to `false`, sync slash commands with Discord on startup.
  Note that unless `COMMANDS_GUILD_ID` is also set, it can take up to an hour for slash commands to show up in the UI.  
  It's recommended to leave this enabled.
- `JOIN_LEAVE_LOG`: a channel ID that the bot will send messages to whenever it joins or leaves a server.
- `SUPPORT_SERVER`: a link to the support server, used in the help command, as well as internal error messages. (not needed for self-hosting)
- `DASHBOARD_BASE`: the base URL to the dashboard. (not needed for self-hosting)
- `SENTRY_URL`: URL to connect to Sentry, for improved error logging (not needed for self-hosting)
- `DEBUG_LOGGING`: if enabled, will enable debug level logging. Not needed for self-hosting, might be useful in development and for reporting errors.
- `INFLUX_URL`: URL to an [InfluxDB](https://www.influxdata.com/products/influxdb-overview/) to report statistics to. (not needed for self-hosting)
- `INFLUX_TOKEN`: Token for InfluxDB (not needed for self-hosting)
- `INFLUX_DB`: Database name to report statistics to. (not needed for self-hosting)

### Dashboard

The dashboard really isn't made with self-hosting in mind; the entire bot can be configured through commands and *securely* setting up the dashboard is too complicated to go into detail here. That being said, it's still possible to run it yourself--but prepare to look through the source code for anything that isn't documented!

To run the dashboard, copy `web/frontend/.env.example` to `web/frontend/.env`, fill it in, build that directory, and run the executable.

You'll also want to set `DASHBOARD_BASE` in the bot's `.env` file. (Optional, but highly recommended, so the `cl!dashboard` command works)

## License

Copyright (C) 2021 Starshine System

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published
by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.

# Logger

A logger bot that integrates with PluralKit's message proxying.  

For a usage guide, check out docs/USAGE.md

To invite the bot to your server, use [this link](https://discord.com/api/oauth2/authorize?client_id=830819903371739166&permissions=537259248&scope=bot%20applications.commands).

## Requirements

- Go 1.16
- PostgreSQL 12

## Installation

1. Create a database
2. Copy `.env.example` to `.env` and fill it in
3. `go build`
4. Run the migrations in `migrations/` (using [tern](https://github.com/jackc/tern))
5. Run the executable

### Dashboard

The dashboard really isn't made with self-hosting in mind; the entire bot can be configured through commands and *securely* setting up the dashboard is too complicated to go into detail here. That being said, the dashboard requires the following extra software:

- Redis, only v6.2 tested
- A reverse proxy

To run the dashboard, copy `web/frontend/.env.example` to `web/frontend/.env`, fill it in, build that directory, and run the executable.

You'll also want to set `DASHBOARD_BASE` in the bot's `.env` file. (Optional, but highly recommended, so the `cl!dashboard` command works)

## License

BSD 3-Clause License

Copyright (c) 2021, Starshine System

All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:

1. Redistributions of source code must retain the above copyright notice, this
   list of conditions and the following disclaimer.

2. Redistributions in binary form must reproduce the above copyright notice,
   this list of conditions and the following disclaimer in the documentation
   and/or other materials provided with the distribution.

3. Neither the name of the copyright holder nor the names of its
   contributors may be used to endorse or promote products derived from
   this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

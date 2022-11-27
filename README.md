# m, a tiny Mastodon service

m is a tiny Mastodon service for folks who want to run their own Mastodon instance, but don't want to run the whole thing.

## What does it do?

m is a tiny Mastodon service that runs on a single server.
It's designed to be run on a small VPS, and it's designed to be run by a single person.

## What _doesn't_ it do?

m doesn't have much of a web interface beyond what ActivityPub requires, you're expected to interact with it via a Mastodon app.

## Getting started

_Warning: m is still in development, if it breaks, you can keep both pieces._

### Pre-requisites

- [Go](https://golang.org/doc/install)
- [MariaDB](https://mariadb.org/download/)

### Installation

Create a database and user for m:

```sql
CREATE DATABASE m;
CREATE USER 'm'@'localhost' IDENTIFIED BY 'm';
GRANT ALL PRIVILEGES ON m.* TO 'm'@'localhost';
```
Install m:

```bash
go install github.com/davecheney/m@latest
```
Create/migrate the database:

```bash
m --debug --dsn 'm:m@/m' auto-migrate
```

### Setup

Create an instance that represents your server:

```bash
m --debug --dsn 'm:m@/m' create-instance --domain example.com
```

Choose wisely, this is permanent.

Create an account for yourself:

```bash
m --debug --dsn 'm:m@/m' create-account --username you --domain example.com --password 🤫
```

This account will be known as `@you@domain.com` in the Fediverse.

### Running

Start m:

```bash
m --debug --dsn 'm:m@/m' serve
```    

### Getting online

m doesn't have a web interface, so you'll need to use a Mastodon app to interact with it.
You'll need to put m behind a reverse proxy, and configure the reverse proxy to forward requests to m.
TLS is also required, so you'll need to configure TLS for your reverse proxy, probably using [Let's Encrypt](https://letsencrypt.org/).

## Acknowledgements 

m would not be possible without these amazing projects

- [Kong](github.com/alecthomas/kong)
- [Gorm](github.com/jinzhu/gorm)

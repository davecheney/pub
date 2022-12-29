# m, a tiny ActivityPub to Mastodon bridge
    
[![Go Reference](https://pkg.go.dev/badge/github.com/davecheney/m.svg)](https://pkg.go.dev/github.com/davecheney/m) [![Go Report Card](https://goreportcard.com/badge/github.com/davecheney/m)](https://goreportcard.com/report/github.com/davecheney/m)
    
## What is m?

m is an ActivityPub host indented for a single actor.
To interact with ActivityPub, m implements the Mastodon[^tm] api for use with various apps. 

m is not intended to host an ActivityPub community, rather it is aimed at enabling someone who owns their own domain, and thus controls their identity, to participate in the fediverse. 

[^tm]: Mastodon is a trademark of [Mastodon gGmbh](https://joinmastodon.org/trademark). m is not affiliated with Mastodon gGmbH. m is not a Mastodon server.

## What _doesn't_ it do?

m doesn't have much of a web interface beyond what ActivityPub requires, you're expected to interact with it via a Mastodon compatible app.

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

Create an account for yourself:

```bash
m --debug --dsn 'm:m@/m' create-account --email you@domain.com --password 🤫 --admin
```

This account will be known as `@you@domain.com` in the Fediverse.

### Running

Start m:

```bash
m --debug --dsn 'm:m@/m' serve --domain domain.com
```    

### Getting online

m doesn't have a web interface, so you'll need to use a Mastodon app to interact with it.
You'll need to put m behind a reverse proxy, and configure the reverse proxy to forward requests to m.
TLS is also required, so you'll need to configure TLS for your reverse proxy, probably using [Let's Encrypt](https://letsencrypt.org/).

## Acknowledgements 

m would not be possible without these amazing projects

- [Kong](github.com/alecthomas/kong)
- [Gorm](github.com/jinzhu/gorm)
- [Chi](github.com/go-chi/chi)

## Contributions

m is open source, but not open for contributions.
That may change in the future, but at the moment please do not send pull requests.
Thank you in advance for your understanding.
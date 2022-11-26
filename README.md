# m, a tiny Mastodon service

m is a tiny Mastodon service.

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
go get github.com/davecheney/m
```
Create/migrate the database:

```bash
m --debug --dsn 'm:m@/m' auto-migrate
```

### Running

Start m:

```bash
m --debug --dsn 'm:m@/m' serve
```    


## Acknowledgements 

m would not be possible without these amazing projects

- github.com/alecthomas/kong
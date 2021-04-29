# d1login

`d1login` implements a login server for Dofus 1.

## Requirements

- [Git](https://git-scm.com/)
- [Go](https://golang.org/)

## Build

```sh
git clone https://github.com/kralamoure/d1login
cd d1login
go build ./cmd/...
```

## Installation

```sh
go get -u -v github.com/kralamoure/d1login/...
```

## Usage

```sh
d1login --help
```

### Output

```text
d1login is a login server for Dofus 1.

Find more information at: https://github.com/kralamoure/d1login

Options:
  -h, --help               Print usage information
  -d, --debug              Enable debug mode
  -a, --address string     Server listener address (default "0.0.0.0:5555")
  -p, --postgres string    PostgreSQL connection string (default "postgresql://user:password@host/database")
  -t, --timeout duration   Connection timeout (default 30m0s)

Usage: d1login [options]
```

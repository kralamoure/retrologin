# retrologin

[![CI](https://github.com/kralamoure/retrologin/actions/workflows/ci.yml/badge.svg)](https://github.com/kralamoure/retrologin/actions/workflows/ci.yml)

`retrologin` is an unofficial login server for Dofus Retro.

## Build

```sh
git clone https://github.com/kralamoure/retrologin
cd retrologin
go build ./cmd/...
```

## Installation

```sh
go install github.com/kralamoure/retrologin/cmd/retrologin@latest
```

## Usage

```sh
retrologin --help
```

### Output

```text
retrologin is an unofficial login server for Dofus Retro.

Find more information at: https://github.com/kralamoure/retrologin

Options:
  -h, --help               Print usage information
  -d, --debug              Enable debug mode
  -a, --address string     Server listener address (default "0.0.0.0:5555")
  -p, --postgres string    PostgreSQL connection string (default "postgresql://user:password@host/database")
  -t, --timeout duration   Connection timeout (default 30m0s)

Usage: retrologin [options]
```

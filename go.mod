module github.com/kralamoure/d1login

go 1.14

require (
	github.com/alexedwards/argon2id v0.0.0-20200802152012-2464efd3196b
	github.com/happybydefault/logger v1.1.0
	github.com/jackc/pgproto3/v2 v2.0.4 // indirect
	github.com/jackc/pgx/v4 v4.8.1
	github.com/kralamoure/d1 v0.0.0-20200811215200-3ff36fd33625
	github.com/kralamoure/d1pg v0.0.0-20200706071528-55530a47673c
	github.com/kralamoure/d1proto v0.0.0-20200713235525-ee4dfe007020
	github.com/kralamoure/dofus v0.0.0-20200812040015-d1ce9c4da9ab
	github.com/kralamoure/dofuspg v0.0.0-20200706071346-573a7477333e
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/crypto v0.0.0-20200728195943-123391ffb6de // indirect
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	golang.org/x/tools v0.0.0-20200519015757-0d0afa43d58a // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)

replace github.com/kralamoure/d1 => ../d1

replace github.com/kralamoure/d1pg => ../d1pg

replace github.com/kralamoure/d1proto => ../d1proto

replace github.com/kralamoure/dofus => ../dofus

replace github.com/kralamoure/dofuspg => ../dofuspg

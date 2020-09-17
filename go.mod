module github.com/kralamoure/d1login

go 1.15

require (
	github.com/alexedwards/argon2id v0.0.0-20200802152012-2464efd3196b
	github.com/happybydefault/logger v1.1.0
	github.com/jackc/pgx/v4 v4.8.1
	github.com/kralamoure/d1 v0.0.0-20200917030335-f23076eacc5c
	github.com/kralamoure/d1pg v0.0.0-20200706071528-55530a47673c
	github.com/kralamoure/d1proto v0.0.0-20200713235525-ee4dfe007020
	github.com/kralamoure/dofus v0.0.0-20200917024449-5e4b76236af8
	github.com/kralamoure/dofuspg v0.0.0-20200917030704-67fe21d1f864
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5
	go.uber.org/atomic v1.7.0
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.16.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	golang.org/x/tools v0.0.0-20200519015757-0d0afa43d58a // indirect
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)

replace github.com/kralamoure/d1 => ../d1

replace github.com/kralamoure/d1pg => ../d1pg

replace github.com/kralamoure/d1proto => ../d1proto

replace github.com/kralamoure/dofus => ../dofus

replace github.com/kralamoure/dofuspg => ../dofuspg

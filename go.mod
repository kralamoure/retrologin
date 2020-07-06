module github.com/kralamoure/d1login

go 1.14

require (
	github.com/alexedwards/argon2id v0.0.0-20200522061839-9369edc04b05
	github.com/happybydefault/logger v1.1.0
	github.com/jackc/pgx/v4 v4.7.1
	github.com/kralamoure/d1 v0.0.0-20200706051325-550660c96ffe
	github.com/kralamoure/d1pg v0.0.0-20200705193926-105845af2c02
	github.com/kralamoure/d1proto v0.0.0-20200701024631-eea21788c6fe
	github.com/kralamoure/dofus v0.0.0-20200705225418-6b7bc89b411c
	github.com/kralamoure/dofuspg v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/time v0.0.0-20200630173020-3af7569d3a1e
	golang.org/x/tools v0.0.0-20200519015757-0d0afa43d58a // indirect
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)

replace github.com/kralamoure/dofus => ../dofus

replace github.com/kralamoure/d1 => ../d1

replace github.com/kralamoure/d1pg => ../d1pg

replace github.com/kralamoure/d1proto => ../d1proto

replace github.com/kralamoure/dofuspg => ../dofuspg

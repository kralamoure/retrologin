module github.com/kralamoure/d1login

go 1.14

require (
	github.com/alexedwards/argon2id v0.0.0-20200522061839-9369edc04b05
	github.com/gofrs/uuid v3.3.0+incompatible
	github.com/kralamoure/d1 v0.0.0-20200623234920-e23803ffa3e1
	github.com/kralamoure/d1postgres v0.0.0-20200621011438-9253ccbff59d
	github.com/kralamoure/d1proto v0.0.0-20200621072541-4ae7bb7f87c4
	github.com/pkg/errors v0.9.1 // indirect
	github.com/spf13/pflag v1.0.5
	go.uber.org/atomic v1.6.0
	go.uber.org/zap v1.15.0
	golang.org/x/lint v0.0.0-20200302205851-738671d3881b // indirect
	golang.org/x/mod v0.3.0 // indirect
	golang.org/x/tools v0.0.0-20200519015757-0d0afa43d58a // indirect
	honnef.co/go/tools v0.0.1-2020.1.4 // indirect
)

replace github.com/kralamoure/d1 => ../d1

replace github.com/kralamoure/d1postgres => ../d1postgres

replace github.com/kralamoure/d1proto => ../d1proto

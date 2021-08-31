module github.com/jorgelbg/pinentry-touchid

go 1.16

replace github.com/foxcpp/go-assuan => ./go-assuan

require (
	github.com/enescakir/emoji v1.0.0 // indirect
	github.com/foxcpp/go-assuan v1.0.0
	github.com/gopasspw/pinentry v0.0.2
	github.com/keybase/go-keychain v0.0.0-20201121013009-976c83ec27a6
	github.com/lox/go-touchid v0.0.0-20170712105233-619cc8e578d0
)

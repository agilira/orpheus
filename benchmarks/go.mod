module github.com/agilira/orpheus/benchmarks

go 1.23.11

toolchain go1.24.5

require (
	github.com/agilira/orpheus v0.0.0
	github.com/alecthomas/kingpin/v2 v2.4.0
	github.com/spf13/cobra v1.8.1
	github.com/urfave/cli/v2 v2.27.4
)

require (
	github.com/agilira/flash-flags v1.0.1 // indirect
	github.com/agilira/go-errors v1.1.0 // indirect
	github.com/alecthomas/units v0.0.0-20211218093645-b94a6e3cc137 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.4 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xhit/go-str2duration/v2 v2.1.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
)

replace github.com/agilira/orpheus => ../

replace github.com/agilira/flash-flags => ../../flash-flags

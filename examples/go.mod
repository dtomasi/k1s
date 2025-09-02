module github.com/dtomasi/k1s/examples

go 1.25.0

require github.com/spf13/cobra v1.8.1

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
)

replace (
	github.com/dtomasi/k1s/core => ../core
	github.com/dtomasi/k1s/storage/memory => ../storage/memory
	github.com/dtomasi/k1s/storage/pebble => ../storage/pebble
)

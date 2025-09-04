module github.com/dtomasi/k1s/tools

go 1.25.0

require github.com/spf13/cobra v1.10.1

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	sigs.k8s.io/controller-tools v0.19.0 // indirect
)

replace github.com/dtomasi/k1s/core => ../core

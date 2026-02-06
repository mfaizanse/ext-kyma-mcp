package main

import (
	"os"

	"k8s.io/cli-runtime/pkg/genericiooptions"

	cmd "github.com/mfaizanse/ext-kyma-mcp/pkg/cmd"
	"github.com/spf13/pflag"
)

func main() {
	flags := pflag.NewFlagSet("kubernetes-mcp-server", pflag.ExitOnError)
	pflag.CommandLine = flags

	root := cmd.NewExtendedMCPServer(genericiooptions.IOStreams{In: os.Stdin, Out: os.Stdout, ErrOut: os.Stderr})
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

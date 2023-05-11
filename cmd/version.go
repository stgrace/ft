package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	// GitCommit is updated with the Git tag by the Goreleaser build
	GitCommit = "unknown"
	// BuildDate is updated with the current ISO timestamp by the Goreleaser build
	BuildDate = "unknown"
	// Version is updated with the latest tag by the Goreleaser build
	Version = "unreleased"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run:   version,
	}
}

func version(_ *cobra.Command, _ []string) {
	fmt.Println("Version:\t", Version)
	fmt.Println("Git commit:\t", GitCommit)
	fmt.Println("Date:\t\t", BuildDate)
	fmt.Println("License:\t Apache 2.0")
}

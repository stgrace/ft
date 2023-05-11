/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ft",
		Short: "Flux test for changes in a remote upstream",
		Long: heredoc.Doc(`Test Flux configuration for changes in

			* helm charts
			* kustomizations
			
			in given directories`),
		// Uncomment the following line if your bare application
		// has an action associated with it:
		// Run: func(cmd *cobra.Command, args []string) { },
	}
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newDiffCmd())
	return cmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := NewRootCmd().Execute()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {

	NewRootCmd().Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

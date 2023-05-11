package cmd

import (
	"fmt"

	flag "github.com/spf13/pflag"

	"github.com/MakeNowJust/heredoc"
	"github.com/helm/chart-testing/v3/pkg/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	chartdiff "github.com/stgrace/ft/pkg/diff"
)

var (
	cfgFile string
)

func newDiffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff",
		Short: "Get HR diff information",
		RunE:  diff,
	}
	flags := cmd.Flags()
	addDiffFlags(flags)
	return cmd

}

func addDiffFlags(flags *flag.FlagSet) {
	flags.StringVar(&cfgFile, "config", "", "Config file")
	flags.String("target-branch", "master", "The name of the target branch used to identify changed charts")
	flags.String("since", "HEAD", "The Git reference used to identify changed charts")
	flags.StringSlice("excluded-charts", []string{}, heredoc.Doc(`
		Charts that should be skipped. May be specified multiple times
		or separate values with commas`))
	flags.StringSlice("chart-dirs", []string{"charts"}, heredoc.Doc(`
		Directories containing Helm charts. May be specified multiple times
		or separate values with commas`))
	flags.Bool("all", false, heredoc.Doc(`
		Process all charts except those explicitly excluded.
		Disables changed charts detection and version increment checking`))
	flags.StringSlice("charts", []string{}, heredoc.Doc(`
		Specific charts to test. Disables changed charts detection and
		version increment checking. May be specified multiple times
		or separate values with commas`))
	flags.StringSlice("chart-repos", []string{}, heredoc.Doc(`
		Additional chart repositories for dependency resolutions.
		Repositories should be formatted as 'name=url' (ex: local=http://127.0.0.1:8879/charts).
		May be specified multiple times or separate values with commas`))
	flags.StringSlice("helm-repo-extra-args", []string{}, heredoc.Doc(`
		Additional arguments for the 'helm repo add' command to be
		specified on a per-repo basis with an equals sign as delimiter
		(e.g. 'myrepo=--username test --password secret'). May be specified
		multiple times or separate values with commas`))
	flags.StringSlice("helm-dependency-extra-args", []string{}, heredoc.Doc(`
		Additional arguments for 'helm dependency build' (e.g. ["--skip-refresh"]`))
	flags.Bool("debug", false, heredoc.Doc(`
		Print CLI calls of external tools to stdout (caution: setting this may
		expose sensitive data when helm-repo-extra-args contains passwords)`))
	flags.Bool("print-config", false, heredoc.Doc(`
		Prints the configuration to stderr (caution: setting this may
		expose sensitive data when helm-repo-extra-args contains passwords)`))
	flags.String("remote", "origin", "The name of the Git remote used to identify changed charts")
}

func diff(cmd *cobra.Command, _ []string) error {
	log.SetLevel(log.DebugLevel)
	log.Info("Checking diff for HelmReleases...")

	printConfig, err := cmd.Flags().GetBool("print-config")
	if err != nil {
		return err
	}
	emptyExtraSetArgs := ""
	configuration, err := config.LoadConfiguration(cfgFile, cmd, printConfig)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}
	testing, err := chartdiff.NewTesting(*configuration, emptyExtraSetArgs)
	if err != nil {
		return err
	}

	results, err := testing.ComputeChangedHelmReleases()
	if err != nil {
		log.Errorf("failed getting HR diff: %s", err)
		return err
	}

	log.Infof("Received results from diff check: %s", results)
	
	return nil
}

package list_github_secrets

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func checkErr(e error) {
	if e != nil {
		fmt.Printf("%v\n", e)
		os.Exit(1)
	}
}

var (
	// Used for flags.
	allRepos      bool
	humanReadable bool
	displayEmpty  bool

	rootCmd = &cobra.Command{
		Use:   "gh-scraper",
		Short: "Tool for managing Cloud Engineering stuff",
		// 		Long: `long weeeeeeeeeeeeeeeeeeeeeeeee`,
	}
)

var listGithubSecretsCmd = &cobra.Command{
	Use:   "list-gh-secrets",
	Short: "print the organisational secrets of the DFDS org, as well as all repo secrets for each repos",
	Run: func(cmd *cobra.Command, args []string) {
		human_readable, err := cmd.Flags().GetBool("human-readable")
		checkErr(err)
		all_repos, err := cmd.Flags().GetBool("display-empty")
		checkErr(err)
		//fmt.Printf("got:  %v as human_readable", human_readable)
		ListghSecretsCmdTest(human_readable, all_repos)
	},
}

package cmd

import (
	"fmt"
	"os"

	list_github_secrets "github.com/dfds/ce-cli/cmd/ghsecrets"
	"github.com/spf13/cobra"
)

var ghSecretsCmd = &cobra.Command{
	Use:   "github",
	Short: "GitHub tooling",
}

func checkErr(e error) {
	if e != nil {
		fmt.Printf("%v\n", e)
		os.Exit(1)
	}
}

var listGithubSecretsCmd = &cobra.Command{
	Use:   "list-gh-secrets",
	Short: "print the organisational secrets of the DFDS org, as well as all repo secrets for each repos",
	Run: func(cmd *cobra.Command, args []string) {
		human_readable, err := cmd.Flags().GetBool("human-readable")
		checkErr(err)
		all_repos, err := cmd.Flags().GetBool("display-empty")
		checkErr(err)
		//fmt.Printf("got:  %v as human_readable", human_readable)
		list_github_secrets.ListghSecretsCmdTest(human_readable, all_repos)
	},
}

func ghSecretsInit() {
	listGithubSecretsCmd.PersistentFlags().Bool("human-readable", false, "human readable output format. defaults to printing json to stdout if not set")
	listGithubSecretsCmd.PersistentFlags().Bool("display-empty", false, "if set: also prints repos with no secrets found")
	rootCmd.AddCommand(listGithubSecretsCmd)
}

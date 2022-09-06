package github

import (
	"context"
	"fmt"
	"github.com/google/go-github/v47/github"
	"github.com/spf13/cobra"
	"go.dfds.cloud/utils/config"
	"golang.org/x/oauth2"
	"log"
	"os"
)

func FixRepoPermissionsCmd(cmd *cobra.Command, args []string) {
	token := config.GetEnvValue("GITHUB_TOKEN", "")
	if token == "" {
		fmt.Println("Missing GITHUB_TOKEN environment variable. Exiting.")
		os.Exit(1)
	}
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var allRepos []*github.Repository
	opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, "dfds", opt)
		if err != nil {
			log.Fatal(err)
		}
		allRepos = append(allRepos, repos...)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	for _, repo := range allRepos {
		fmt.Println(*repo.Name)
		_, err := client.Teams.AddTeamRepoBySlug(ctx, "dfds", "cloud-engineering", "dfds", *repo.Name, &github.TeamAddTeamRepoOptions{Permission: "admin"})
		if err != nil {
			log.Fatal(err)
		}
	}
}

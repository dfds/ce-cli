package list_github_secrets

//functions which handle the calls to the github API Go SDK

import (
	"context"
	"fmt"
	"os"
	"sync"

	//"github.com/google/go-github/v49/github"
	"github.com/google/go-github/v47/github"
	"golang.org/x/oauth2"
)

func get_token() string {
	return os.Getenv("GITHUB_API_TOKEN")
}

func make_gh_client() *github.Client {
	// return an authenticated github SDK-
	// -http client.
	// use token got from devex-sa account PATs
	token := get_token()
	ctx := context.Background()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)
	return client
}

func gh_util_vars() (context.Context, *github.Client, *github.RepositoryListByOrgOptions) {
	// return "defaults" for objects needed
	// by the github SDK methods.
	// an actions client covers all needs in this package
	gh_opt := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	ctx := context.Background()
	client := make_gh_client()

	return ctx, client, gh_opt
}

func ListRepos(token string, repoMap *repoIdentifier) (*repoIdentifier, error) {
	// receives a ref to a repoIdentifier struct and
	// fills its maps with values to lookup repo
	// objects for all repos in the DFDS organization
	var repo_info_list []*github.Repository

	ctx, client, gh_opt := gh_util_vars()

	var err error

	for {
		repos, resp, err := client.Repositories.ListByOrg(ctx, "dfds", gh_opt)
		if err != nil {
			//TODO: handle fail
			fmt.Println(err)
			os.Exit(1)
		}
		repo_info_list = append(repo_info_list, repos...)
		if resp.NextPage == 0 {
			break
		}
		gh_opt.Page = resp.NextPage
	}
	for _, r := range repo_info_list {
		repoMap.byId[*r.ID] = *r
		repoMap.byName[*r.Name] = *r
	}
	return &m_repo_Identifiers, err //repo_names, //repo_ids, err, &m_repo_Identifiers
}

func GetRepoEnvironments(reponame string) ([]*github.Environment, error) {
	// get all repo environment objects for a given repo

	ctx, gh_client, _ := gh_util_vars()
	gh_opt := &github.EnvironmentListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	envs, _, err := gh_client.Repositories.ListEnvironments(ctx, "dfds", reponame, gh_opt)
	return envs.Environments, err
}

func getRepoSecrets(dest syncRepoMap, repomap *repoIdentifier) error {
	//takes a pointer to a map reponame->repoSecret
	//takes a reference to a repoIdentifier, loops over one of its maps.
	//populates the given map concurrently with each repo found and its secrets
	ctx, gh_client, gh_opt := gh_util_vars()
	var err error
	var wg *sync.WaitGroup = &sync.WaitGroup{}
	//var did []string
	for reponame := range repomap.byName {
		//did = append(did, reponame)// for debugging
		wg.Add(1)
		go func(inner *sync.WaitGroup, rnameInner string) {
			defer inner.Done()
			secrets, _, err := gh_client.Actions.ListRepoSecrets(ctx, "dfds", rnameInner, &gh_opt.ListOptions)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			if len(secrets.Secrets) > 0 {
				for _, secret := range secrets.Secrets {
					curr_secret := repoSecret{
						secret.Name,
						secret.UpdatedAt,
						secret.Visibility,
					}
					dest.s.Lock()
					dest.m[rnameInner] = append(dest.m[rnameInner], curr_secret)
					dest.s.Unlock()
				}
			} else {
				dest.s.Lock()
				dest.m[rnameInner] = []repoSecret{}
				dest.s.Unlock()
			}
		}(wg, reponame)
	}
	wg.Wait()
	return err
}

func getEnvSecrets(reponame, env string, repoID int64) (*github.Secrets, error) {
	//get secrets field of a github environment secret
	ctx, gh_client, gh_opt := gh_util_vars()
	secrets, _, err := gh_client.Actions.ListEnvSecrets(ctx, int(repoID), env, &gh_opt.ListOptions)
	return secrets, err
}

func getOrgSecrets() (map[string]orgSecret, error) {
	//get org secrets and return a map of where they're being used
	// as well as a map from secret IDs to their last updated timestamp
	var wg *sync.WaitGroup = &sync.WaitGroup{}
	ctx, gh_client, gh_opt := gh_util_vars()
	//secrets2repos := make(map[string][]string)
	secretsInfo := syncOrgSecretsMap{m: make(map[string]orgSecret), s: &sync.Mutex{}}
	//secretsLastUpdated := make(map[string]github.Timestamp)
	var err error

	secrets, _, err := gh_client.Actions.ListOrgSecrets(ctx, "dfds", &gh_opt.ListOptions)

	var selectedRepos *github.SelectedReposList
	for _, secret := range secrets.Secrets {
		wg.Add(1)
		go func(inner *sync.WaitGroup, secretInner *github.Secret) {
			defer inner.Done()
			selectedRepos, _, err = gh_client.Actions.ListSelectedReposForOrgSecret(ctx, "dfds", secretInner.Name, &gh_opt.ListOptions)

			var tmp_repo_list []string
			for _, repo := range selectedRepos.Repositories {
				tmp_repo_list = append(tmp_repo_list, *repo.Name)
			}
			secretsInfo.s.Lock()
			secretsInfo.m[secretInner.Name] = orgSecret{
				secretInner.Name,
				secretInner.UpdatedAt,
				secretInner.Visibility,
				tmp_repo_list,
			}
			secretsInfo.s.Unlock()

		}(wg, secret)
	}
	wg.Wait()
	return secretsInfo.m, err
}

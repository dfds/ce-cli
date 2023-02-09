package list_github_secrets

import "fmt"

func HumanReadableOrgSecret(secret orgSecret) string {
	s := fmt.Sprintf("\n %v [Last updated: %v]\n", secret.Name, secret.LastUpdated)
	if len(secret.Used_in) > 0 {
		for _, repo := range secret.Used_in {
			s += fmt.Sprintf("+ used in repository %v\n", repo)
		}
	} else {
		s += "- no repos found using this secret \n"
	}
	return s
}

func HumanReadableRepoSecret(secret repoSecret) string {
	return fmt.Sprintf("+ [%v]%v, last updated [%v]\n", secret.Name, secret.sort, secret.LastUpdated)
}

func hReadableRepos(repos map[string][]repoSecret, displayEmpty bool) string {
	var s string
	s += "\n################\n# repo secrets #\n################\n"
	for reponame, secrets := range repos {
		if len(secrets) > 0 {
			s += fmt.Sprintf("\n repo [%v] has\n", reponame)
			for _, sec := range secrets {
				s += HumanReadableRepoSecret(sec)
			}
		}
		if len(secrets) == 0 && displayEmpty {
			s += fmt.Sprintf("\n repo [%v] has\n", reponame)
			s += "- no secrets\n"
		}
	}

	return s
}

func hReadableOrgs(secrets map[string]orgSecret) string {
	var s string
	s += "\n################\n# org  secrets #\n################\n"
	for _, sec := range secrets {
		s += HumanReadableOrgSecret(sec)
	}
	return s
}

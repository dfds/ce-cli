package list_github_secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/go-github/v47/github" /*we know it works with v47*/
)

type repoIdentifier struct { // we assume the number of repos will always be ~O(10^2)
	byName map[string]github.Repository
	byId   map[int64]github.Repository
}

type syncRepoMap struct { //repo map with mutex a thread can lock
	m map[string][]repoSecret
	s *sync.Mutex
}

type syncOrgSecretsMap struct {
	m map[string]orgSecret
	s *sync.Mutex
}

func NewRepoIdentifier() repoIdentifier {
	byName := make(map[string]github.Repository)
	byid := make(map[int64]github.Repository)
	rid := repoIdentifier{byName, byid}

	return rid
}

var m_repo_Identifiers = NewRepoIdentifier()

func unmarshalResponseAny(b []byte, res interface{}) error {
	// unmarshal a json response to any struct
	// assumes you know what the body of your
	// response `b` will look like.
	return json.Unmarshal([]byte(b), &res)
}

func ListghSecretsCmdTest(humanReadable, displayEmpty bool) {
	repo2secretsmap := syncRepoMap{m: make(map[string][]repoSecret), s: &sync.Mutex{}}
	//setting utility variables
	//wantverbose := false
	token := get_token()
	start := time.Now()
	_, err := ListRepos(token, &m_repo_Identifiers)
	if err != nil {
		fmt.Printf("%v\n", err)
	}

	//Org secrets
	m, err := getOrgSecrets()
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	if humanReadable {
		s := hReadableOrgs(m)
		fmt.Print(s)
	} else {
		jsonString, err := json.Marshal(m)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("%s\n", jsonString)
	}

	//repo secrets
	err = getRepoSecrets(repo2secretsmap, &m_repo_Identifiers)
	if err != nil {
		fmt.Print(err)
		os.Exit(1)
	}
	if humanReadable {
		s := hReadableRepos(repo2secretsmap.m, displayEmpty)
		fmt.Print(s)
	} else {
		jsonString, err := json.Marshal(repo2secretsmap.m)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf("%s\n", jsonString)
	}

	elapsed := time.Since(start)
	if humanReadable {
		fmt.Printf("\n ran in: %v\n", elapsed)
	}
}

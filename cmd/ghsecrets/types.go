package list_github_secrets

import "github.com/google/go-github/v47/github"

type orgSecret struct {
	Name        string
	LastUpdated github.Timestamp
	Visibility  string
	Used_in     []string
}

//these structs are not really
// different in terms of the info they contain
// they just have different names for readbility of the code

type repoSecret struct {
	Name        string
	LastUpdated github.Timestamp
	sort        string
}

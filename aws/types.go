package aws

type roleProperties struct {
	Description     string   `json:"description"`
	SessionDuration int32    `json:"sessionDuration"`
	Path            string   `json:"path"`
	ManagedPolicies []string `json:"managedpolicies"`
}

type excludeAccountsStruct struct {
	Scopes excludeAccountsScopes `json:"Scopes"`
}

type excludeAccountsScopes struct {
	Common       []string `json:"Common"`
	ListAccounts []string `json:"ListAccounts"`
	OidcProvider []string `json:"OidcProvider"`
}

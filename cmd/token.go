package cmd

import (
	"bytes"
	"fmt"
	rice "github.com/GeertJohan/go.rice"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Retrieve token for authenticating",
	Run: func(cmd *cobra.Command, args []string) {
		tokenCmdFunc(cmd, args)
	},
}

func tokenInit() {
	tokenCmd.PersistentFlags().StringP("app", "a", "", "Pick one of the predefined apps: 'selfservice-portal'")
	tokenCmd.PersistentFlags().StringP("tenant", "t", "", "If not using a predefined app, specify a tenant id")
	tokenCmd.PersistentFlags().StringP("scope", "s", "", "If not using a predefined app, specify scopes, e,g. 'openid,profile'")
	tokenCmd.PersistentFlags().String("app-id", "", "If not using a predefined app, specify scopes, e,g. 'openid,profile'")

}

const defaultTenant = "73a99466-ad05-4221-9f90-e7142aa2f6c1"

func tokenCmdFunc(cmd *cobra.Command, args []string) {
	options := initTokenOptions()

	appInput, _ := cmd.Flags().GetString("app")
	tenantInput, _ := cmd.Flags().GetString("tenant")
	scopeInput, _ := cmd.Flags().GetString("scope")
	appIdInput, _ := cmd.Flags().GetString("app-id")

	if tenantInput == "" {
		tenantInput = defaultTenant
	}

	if val, ok := options[appInput]; ok {
		scopeInput = val.Scope
		appIdInput = val.AppId
	} else {
		if appInput != "" {
			fmt.Println("--app value doesn't exist")
			os.Exit(1)
		}
	}

	urlParsed, err := url.Parse(fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenantInput))
	if err != nil {
		log.Fatal(err)
	}

	values := urlParsed.Query()
	values.Set("client_id", appIdInput)
	values.Set("response_type", "token")
	values.Set("scope", scopeInput)
	values.Set("nonce", "12345")
	values.Set("redirect_uri", "http://localhost:4200/login")
	urlParsed.RawQuery = values.Encode()

	browser.OpenURL(urlParsed.String())
	responseServer()
}

type tokenOptions struct {
	Scope string
	AppId string
}

func initTokenOptions() map[string]tokenOptions {
	options := make(map[string]tokenOptions)

	options["selfservice-portal"] = tokenOptions{
		Scope: "api://3007f683-c3c2-4bf9-b6bd-2af03fb94f6d/.default",
		AppId: "3007f683-c3c2-4bf9-b6bd-2af03fb94f6d",
	}

	return options
}

func responseServer() {
	http.HandleFunc("/", serveFile)
	http.HandleFunc("/success", successHandler)
	http.ListenAndServe(":4200", nil)
}

func successHandler(resp http.ResponseWriter, req *http.Request) {
	box := rice.MustFindBox("token")
	httpBox := box.HTTPBox()
	templateData, err := httpBox.Bytes("success.html")
	if err != nil {
		log.Fatal(err)
	}

	err = req.ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	templateContainer := template.New("gen")
	templateParsed, err := templateContainer.Parse(string(templateData))
	if err != nil {
		log.Fatal(err)
	}

	var body bytes.Buffer
	templateParsed.Execute(&body, templateVars{
		Token: req.Form.Get("token"),
	})

	resp.WriteHeader(200)
	resp.Write(body.Bytes())

	go func() {
		time.Sleep(time.Second * 1)
		os.Exit(0)
	}()
}

func serveFile(resp http.ResponseWriter, req *http.Request) {
	box := rice.MustFindBox("token")
	httpBox := box.HTTPBox()
	path, _ := strings.CutPrefix(req.URL.Path, "/")

	// Silly, but can't be bothered to add anything more sophisticated given the volume.
	if path == "login" {
		path = "login.html"
	}

	buf, err := httpBox.Bytes(path)
	if err != nil {
		resp.WriteHeader(400)
		return
	}

	resp.WriteHeader(200)
	resp.Write(buf)
}

type templateVars struct {
	Token string
}

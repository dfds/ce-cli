package k8s

import (
	"fmt"
	cp "github.com/otiai10/copy"
	"log"
	"os"
	"os/exec"
)

type GenerateConfigRequest struct {
	ClusterName string
}

const templateDirPath = ".k8s-sandbox-template"
const templateRemoteRepo = "git@github.com:dfds/eks-pipeline.git"

func DownloadTemplateRepo(path string) error {
	// Check if a previous template directory hasn't been cleaned up. If that's the case, remove it.
	if checkIfDirExists(path) {
		err := os.RemoveAll(path)
		if err != nil {
			return err
		}
	}

	cmd := exec.Command("git", "clone", templateRemoteRepo, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return err
	}

	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("Can't find the downloaded template directory. Unable to continue.")
		}
		log.Fatal(err)
	}

	return nil
}

func GenerateConfig(req GenerateConfigRequest) error {
	//var templateDir = fmt.Sprintf("%s/oxygen-account/eu-west-1/k8s-hellman/", templateDirPath)
	var sandboxDir = fmt.Sprintf("clusters/k8s-%s", req.ClusterName)
	//var sandboxClusterManifest = fmt.Sprintf("%s/cluster/terragrunt.hcl")
	//var sandboxServicesManifest = fmt.Sprintf("%s/services/terragrunt.hcl")

	err := DownloadTemplateRepo(templateDirPath)
	if err != nil {
		return err
	}

	err = cp.Copy(fmt.Sprintf("%s/oxygen-account", templateDirPath), sandboxDir)
	if err != nil {
		return err
	}

	return nil
}

func checkIfDirExists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false
		}
	}
	return true
}

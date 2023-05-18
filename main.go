package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

var (
	GithubToken = os.Getenv("GITHUB_TOKEN")
	Directory   = "./crds/"
)

func main() {
	client := newGithubClient(GithubToken)
	ctx := context.Background()

	ackRepositories, _, err := client.Repositories.ListByOrg(
		ctx,
		"aws-controllers-k8s",
		&github.RepositoryListByOrgOptions{
			ListOptions: github.ListOptions{PerPage: 200},
		})
	if err != nil {
		log.Panic(err.Error())
	}

	// Get controller names
	controllersNames := []string{}
	for _, repository := range ackRepositories {
		if repoIsController(*repository.Name) {
			controllersNames = append(controllersNames, *repository.Name)
		}
	}
	log.Printf("found %d controller repositories\ndownloading CRDs...\n", len(controllersNames))
	totalsCRDs := 0

	// get and write CRDs
	for _, name := range controllersNames {
		_, dirContent, _, err := client.Repositories.GetContents(
			ctx,
			"aws-controllers-k8s",
			name,
			"config/crd/bases",
			nil,
		)
		if err != nil {
			// some controllers don't have files yet..
			// log.Panic(err.Error())
		}

		for _, crdFile := range dirContent {
			totalsCRDs++
			log.Printf("[repository=%s] Getting crd file: %s\n", name, *crdFile.Name)
			crdContent, err := http.Get(*crdFile.DownloadURL)
			if err != nil {
				log.Panic(err.Error())
			}
			b, err := io.ReadAll(crdContent.Body)
			if err != nil {
				log.Panic(err.Error())
			}
			err = os.WriteFile(Directory+*crdFile.Name, b, 0644)
			if err != nil {
				log.Panic(err.Error())
			}
		}
	}
	log.Printf("Total downloaded CRDs: %d\n", totalsCRDs)

}

// newGithubClient takes a token and instantiate a new Client object
func newGithubClient(token string) *github.Client {
	ctx := context.TODO()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oc := oauth2.NewClient(ctx, ts)
	return github.NewClient(oc)
}

func repoIsController(s string) bool {
	return strings.HasSuffix(s, "-controller")
}

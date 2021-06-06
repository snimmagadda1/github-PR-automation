package client

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
	v3 "github.com/google/go-github/v35/github"
	"github.com/snimmagadda1/github-PR-automation/config"
)

type GithubService struct {
	client *config.GithubClient
	atr    *ghinstallation.AppsTransport // TODO: What is the difference
	itr    *ghinstallation.Transport
}

// MewGithubService instantiates a transport based on passed configs
func NewGithubService(config *config.GithubClient) (*GithubService, error) {
	service := &GithubService{
		client: config,
	}

	err := service.initService()
	if err != nil {
		return nil, err
	}

	return service, nil
}

func (s *GithubService) initService() error {
	// Create an app transport (semi-authenticated)
	atr, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, s.client.AppID, s.client.PrivateKeyCertPath)
	if err != nil {
		log.Fatal("error creating GitHub app client", err)
	}

	s.atr = atr

	if s.client.Enterprise {
		client, err := v3.NewEnterpriseClient(s.client.GitHubURL, s.client.GithubUploadURL, &http.Client{Transport: s.atr})
		if err != nil {
			log.Fatal("failed to init enterprise client", err)
		}

		// Extra step for org install (owner=orgId if using enterprise)
		installation, _, err := client.Apps.FindOrganizationInstallation(context.TODO(), s.client.Owner)
		if err != nil {
			log.Fatalf("error finding organization installation: %v", err)
		}

		installationID := installation.GetID()
		itr := ghinstallation.NewFromAppsTransport(atr, installationID)
		itr.BaseURL = s.client.GitHubURL
		s.itr = itr
		log.Printf("successfully initialized enterprise GitHub app client url:%s, installation-id:%d expected-events:%v\n", itr.BaseURL, installationID, installation.Events)
	}

	return nil
}

// GetV3Client returns a githb authorized client
// TODO: might be more efficient way to handle non-org case w/ installationID only avail at event receieved
func (s *GithubService) GetV3Client(installationID int) *v3.Client {
	if s.client.Enterprise {
		client, err := v3.NewEnterpriseClient(s.client.GitHubURL, s.client.GithubUploadURL, &http.Client{Transport: s.itr})
		if err != nil {
			log.Fatal("failed to generate a client", err)
		}
		return client
	} else {
		// Non-enterprise must authenticate as installation individually based on repo
		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, s.client.AppID, int64(installationID), s.client.PrivateKeyCertPath)
		if err != nil {
			log.Fatal("failed to generate a client", err)
		}
		return v3.NewClient(&http.Client{Transport: itr})
	}
}

// GetRef returns the commit branch reference object if it exists or creates it
// from the base branch before returning it. From https://github.com/google/go-github/blob/master/example/commitpr/main.go
func (s *GithubService) GetRef(installationID int, repo string, baseBranch string, commitBranch string) (ref *v3.Reference, err error) {
	if ref, _, err = s.GetV3Client(installationID).Git.GetRef(context.TODO(), s.client.Owner, repo, "refs/heads/"+commitBranch); err == nil {
		return ref, nil
	}

	// We consider that an error means the branch has not been found and needs to
	// be created.
	if commitBranch == baseBranch {
		return nil, errors.New("The commit branch does not exist but `-base-branch` is the same as `-commit-branch`")
	}

	if baseBranch == "" {
		return nil, errors.New("The `-base-branch` should not be set to an empty string when the branch specified by `-commit-branch` does not exists")
	}

	var baseRef *v3.Reference
	if baseRef, _, err = s.GetV3Client(installationID).Git.GetRef(context.TODO(), s.client.Owner, repo, "refs/heads/"+baseBranch); err != nil {
		return nil, err
	}
	newRef := &v3.Reference{Ref: v3.String("refs/heads/" + commitBranch), Object: &v3.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = s.GetV3Client(installationID).Git.CreateRef(context.TODO(), s.client.Owner, repo, newRef)
	return ref, err
}

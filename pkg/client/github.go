package client

import (
	"context"
	"errors"
	"io/ioutil"
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

	var baseRef *v3.Reference
	if baseRef, _, err = s.GetV3Client(installationID).Git.GetRef(context.TODO(), s.client.Owner, repo, "refs/heads/"+baseBranch); err != nil {
		return nil, err
	}

	// If existing ref for merge branch, update the ref with latest changes and return
	if ref, _, err = s.GetV3Client(installationID).Git.GetRef(context.TODO(), s.client.Owner, repo, "refs/heads/"+commitBranch); err == nil {
		log.Printf("Found stale merge branch with ref %s with hash %s. UPDATING", ref.GetRef(), ref.GetObject().GetSHA())
		newRef := &v3.Reference{Ref: v3.String("refs/heads/" + commitBranch), Object: &v3.GitObject{SHA: baseRef.Object.SHA}}
		ref, res, err := s.GetV3Client(installationID).Git.UpdateRef(context.TODO(), s.client.Owner, repo, newRef, true)
		if err != nil {
			body, _ := ioutil.ReadAll(res.Body)
			bodyString := string(body)
			return nil, errors.New("Detected stale merge branch however an error occurred during update. Reason: " + bodyString)
		}

		log.Printf("Successfully updated existing merge branch. If a pull request already exists, creation may fail but updates should be reflected")
		return ref, nil
	}

	if commitBranch == baseBranch {
		return nil, errors.New("The commit branch does not exist but `-base-branch` is the same as `-commit-branch`")
	}

	if baseBranch == "" {
		return nil, errors.New("The `-base-branch` should not be set to an empty string when the branch specified by `-commit-branch` does not exists")
	}

	newRef := &v3.Reference{Ref: v3.String("refs/heads/" + commitBranch), Object: &v3.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = s.GetV3Client(installationID).Git.CreateRef(context.TODO(), s.client.Owner, repo, newRef)
	return ref, err
}

// AssignRevs assigns reviewers by iterating over recent commits and adding authors
func (s *GithubService) AssignRevs(installationID int, repo string, pr *v3.PullRequest) (err error) {
	comts, _, err := s.GetV3Client(installationID).PullRequests.ListCommits(context.TODO(), s.client.Owner, repo, *pr.Number, nil)
	if err != nil {
		return errors.New("Unable to list commits needed to obtain reviewers for PR: " + *pr.Title)
	}

	// Iterate arbitrary num previous committers
	auths := make([]string, 6)
	var count int = 0
	for _, comt := range comts {
		auths = append(auths, *comt.Committer.Login)
		count++
	}

	// Add reviewers
	_, _, err = s.GetV3Client(installationID).PullRequests.RequestReviewers(context.TODO(), s.client.Owner, repo, *pr.Number, v3.ReviewersRequest{
		Reviewers: auths,
	})
	if err != nil {
		return errors.New("Unable to add reviewrs for PR: " + *pr.Title)
	}

	return nil
}

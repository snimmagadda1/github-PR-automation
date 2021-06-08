package main

import (
	"context"
	"log" // TODO: Replace w/ more robust
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	v3 "github.com/google/go-github/v35/github"
	"github.com/joho/godotenv"
	"github.com/snimmagadda1/github-PR-automation/config"
	"github.com/snimmagadda1/github-PR-automation/pkg/client"
	"github.com/snimmagadda1/github-PR-automation/pkg/utils"
	ghwebhooks "gopkg.in/go-playground/webhooks.v5/github"
)

// Env based config for now
var (
	webhookSecret             = "development"
	appID                     int64
	owner                     string
	useEnterprise             bool
	GitHubEnterpriseURL       string
	GitHubEnterpriseUploadURL string
	certPath                  string
	masterBranch              string
	releaseBranch             string
	repos                     []string // might be better as map
	s                         *client.GithubService
)

func processEvent(p *ghwebhooks.PushPayload) {
	// TODO: More robust comparison logic here
	isRelease := strings.Contains(strings.ToLower(p.Ref), strings.ToLower(releaseBranch)) && !strings.Contains(p.Ref, "merge")
	if !isRelease {
		log.Printf("Push event on non-release branch: %s. Ignoring", p.Ref)
		return
	}

	if repo := p.Repository.Name; utils.Contains(repos, repo) {
		// Check out new branch of main
		mergeBranch := "merge-" + releaseBranch
		ref, err := s.GetRef(p.Installation.ID, repo, releaseBranch, mergeBranch)
		if err != nil {
			log.Fatalf("Unable to get/create the commit reference: %s\n", err)
		}

		if ref == nil {
			log.Fatalf("No error where returned but the reference is nil")
		}

		// Create PR on new branch
		pr, _, err := s.GetV3Client(p.Installation.ID).PullRequests.Create(context.TODO(), owner, repo, &v3.NewPullRequest{
			Title:               v3.String("Merge " + releaseBranch),
			Head:                v3.String(strings.ToLower(mergeBranch)),
			Base:                v3.String(masterBranch),
			Body:                v3.String("This is an automatically created PR 🚀"),
			MaintainerCanModify: v3.Bool(true),
		})
		if err != nil {
			log.Printf("Unable to create pull request. Reason: %v", err)
		} else {
			log.Printf("created pull request: %s", pr.GetURL())
		}
	} else {
		log.Printf("parsed push - unmonitored repo: %s", repo)
	}
}

func Handle(response http.ResponseWriter, request *http.Request) {
	hook, err := ghwebhooks.New(ghwebhooks.Options.Secret(webhookSecret))
	if err != nil {
		return
	}

	payload, err := hook.Parse(request, []ghwebhooks.Event{ghwebhooks.PushEvent}...)
	if err != nil {
		if err == ghwebhooks.ErrEventNotFound {
			log.Printf("received unregistered GitHub event: %v\n", err)
			response.WriteHeader(http.StatusOK)
		} else {
			log.Printf("received malformed GitHub event: %v\n", err)
			response.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	switch payload := payload.(type) {
	case ghwebhooks.PushPayload:
		log.Println("received push event")
		// handle async b/c github wants speedy replies
		go processEvent(&payload)
	default:
		log.Println("missing handler")
		log.Printf("receieved release payload of type %v", payload)
	}

	response.WriteHeader(http.StatusOK)
}

func init() {
	// loads values from .env into the system
	if err := godotenv.Load(); err != nil {
		log.Fatalf("No .env file found")
	}

	stringAppId := os.Getenv("APP_ID")
	if intAppId, err := strconv.Atoi(stringAppId); err == nil {
		appID = int64(intAppId)
	} else {
		log.Fatalf("Could not parse appId: %v", err)
	}

	owner = os.Getenv("OWNER")
	GitHubEnterpriseURL = os.Getenv("GITHUB_ENTERPRISE_URL")
	useEnterprise = false
	if GitHubEnterpriseURL != "" {
		useEnterprise = true
	}
	GitHubEnterpriseUploadURL = os.Getenv("GITHUB_ENTERPRISE_UPLOAD_URL")
	certPath = os.Getenv("CERT_PATH")
	releaseBranch = os.Getenv("RELEASE_BRANCH")
	masterBranch = os.Getenv("MASTER_BRANCH")
	repos = utils.GetEnvAsSlice(os.Getenv("REPOS"), ",")
	sort.Strings(repos)
	log.Printf("Initialized environment with appId: %d, owner: %s, certPath: %s, enterpriseUrl: %s, enterpriseUploadUrl: %s, releaseBranch: %s, repos: %v", appID, owner, certPath, GitHubEnterpriseURL, GitHubEnterpriseUploadURL, releaseBranch, repos)
}

func main() {
	// Create config
	clientConfig := &config.GithubClient{
		AppID:              appID,
		Owner:              owner,
		Enterprise:         useEnterprise,
		GitHubURL:          GitHubEnterpriseURL,
		GithubUploadURL:    GitHubEnterpriseUploadURL,
		PrivateKeyCertPath: certPath,
	}

	// init client
	serv, err := client.NewGithubService(clientConfig)
	if err != nil {
		log.Fatalf("Failed to create client service: %v", err)
	}
	s = serv

	// handle
	http.HandleFunc("/", Handle)
	log.Print("Ready to handle github events")
	err = http.ListenAndServe("0.0.0.0:3000", nil)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

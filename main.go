package main

import (
	"context"
	"log" // TODO: Replace w/ more robust
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/bradleyfalzon/ghinstallation"
	v3 "github.com/google/go-github/v35/github"
	"github.com/joho/godotenv"
	ghwebhooks "gopkg.in/go-playground/webhooks.v5/github"
)

// Env based config for now
var (
	webhookSecret             = "development"
	appID                     int64
	orgID                     string
	useEnterprise             bool
	GitHubEnterpriseURL       string
	GitHubEnterpriseUploadURL string
	certPath                  string
	releaseBranch             string
	repos                     []string // might be better as map
	installationID            int64
	itr                       *ghinstallation.Transport
)

func GetV3Client() *v3.Client {
	if useEnterprise {
		client, err := v3.NewEnterpriseClient(GitHubEnterpriseURL, GitHubEnterpriseUploadURL, &http.Client{Transport: itr})
		if err != nil {
			log.Fatal("failed to generate a client", err)
		}
		return client
	} else {
		return v3.NewClient(&http.Client{Transport: itr})
	}
}

func contains(s []string, searchterm string) bool {
	i := sort.SearchStrings(s, searchterm)
	contains := i < len(s) && s[i] == searchterm
	return contains
}

func processReleaseEvent(p *ghwebhooks.PushPayload) {
	isRelease := strings.Contains(strings.ToLower(p.Ref), strings.ToLower(releaseBranch))
	if isRelease {
		if branch := p.Repository.Name; contains(repos, branch) {
			pr, _, err := GetV3Client().PullRequests.Create(context.TODO(), orgID, branch, &v3.NewPullRequest{
				Title:               v3.String("Merge " + releaseBranch),
				Head:                v3.String(strings.ToLower(releaseBranch)),
				Base:                v3.String("master"),
				Body:                v3.String("This is an automatically created PR 🚀"),
				MaintainerCanModify: v3.Bool(true),
			})
			if err != nil {
				if !strings.Contains(err.Error(), "A pull request already exists") {
					log.Printf("error creating pull request: %v\n", err)
				}
			} else {
				log.Printf("created pull request: %s", pr.GetURL())
			}
		} else {
			log.Printf("parsed push - unmonitored repo: %s", branch)
		}
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
		go processReleaseEvent(&payload)
	default:
		log.Println("missing handler")
		log.Printf("receieved release payload of type %v", payload)
	}

	response.WriteHeader(http.StatusOK)
}

func getEnvAsSlice(in string, sep string) []string {
	val := strings.Split(in, sep)

	return val
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

	orgID = os.Getenv("ORG_ID")
	GitHubEnterpriseURL = os.Getenv("GITHUB_ENTERPRISE_URL")
	if GitHubEnterpriseURL != "" {
		useEnterprise = true
	}

	GitHubEnterpriseUploadURL = os.Getenv("GITHUB_ENTERPRISE_UPLOAD_URL")
	certPath = os.Getenv("CERT_PATH")
	releaseBranch = os.Getenv("RELEASE_BRANCH")
	repos = getEnvAsSlice(os.Getenv("REPOS"), ",")
	sort.Strings(repos)
	log.Printf("Initialized environment with appId: %d, orgId: %s, certPath: %s, enterpriseUrl: %s, enterpriseUploadUrl: %s, releaseBranch: %s, repos: %v", appID, orgID, certPath, GitHubEnterpriseURL, GitHubEnterpriseUploadURL, releaseBranch, repos)
}

func main() {

	atr, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, appID, certPath)
	if err != nil {
		log.Fatal("error creating GitHub app client", err)
	}

	var client *v3.Client
	if useEnterprise {
		client, err = v3.NewEnterpriseClient(GitHubEnterpriseURL, GitHubEnterpriseUploadURL, &http.Client{Transport: atr})
		if err != nil {
			log.Fatal("failed to init client", err)
		}
	} else {
		client = v3.NewClient(&http.Client{Transport: itr})
	}

	installation, _, err := client.Apps.FindOrganizationInstallation(context.TODO(), orgID)
	if err != nil {
		log.Fatalf("error finding organization installation: %v", err)
	}

	installationID = installation.GetID()
	itr = ghinstallation.NewFromAppsTransport(atr, installationID)
	itr.BaseURL = GitHubEnterpriseURL

	log.Printf("successfully initialized GitHub app client url:%s, installation-id:%d expected-events:%v\n", itr.BaseURL, installationID, installation.Events)

	http.HandleFunc("/", Handle)
	err = http.ListenAndServe("0.0.0.0:3000", nil)
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

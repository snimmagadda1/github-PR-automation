package client

import (
	"log"
	"net/http"

	"github.com/bradleyfalzon/ghinstallation"
)

func GetV3Client(enterprise bool, installationID int) *v3.Client {
	if enterprise {
		client, err := v3.NewEnterpriseClient(GitHubEnterpriseURL, GitHubEnterpriseUploadURL, &http.Client{Transport: itr})
		if err != nil {
			log.Fatal("failed to generate a client", err)
		}
		return client
	}
	// Non-enterprise must authenticate as installation individually
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, appID, int64(installationID), certPath)
	if err != nil {
		log.Fatal("failed to generate a client", err)
	}
	return v3.NewClient(&http.Client{Transport: itr})

}

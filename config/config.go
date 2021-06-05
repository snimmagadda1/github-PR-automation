package config

type GithubClient struct {
	AppID              int64
	Owner              string
	Enterprise         bool
	GitHubURL          string
	GithubUploadURL    string
	PrivateKeyCertPath string
}

package config

type GithubClient struct {
	AppID              int64
	Owner              string
	Enterprise         bool
	PrivateKeyCertPath string
}

type Client struct {
	Client *GithubClient `json:"client" description:"authorized github client"`
	Owner  int64         `json:"owner" description:"OwnerId or OrgId"`
}

package settings

import "strings"

// DeployRepo configures install of goscan-remote from a private git repository.
type DeployRepo struct {
	URL    string `yaml:"url,omitempty"`
	Ref    string `yaml:"ref,omitempty"`    // empty → tag v{localVersion}
	Token  string `yaml:"token,omitempty"`  // HTTPS private repos only
	Method string `yaml:"method,omitempty"` // git | sftp — default git when URL set
}

const (
	DeployGit  = "git"
	DeploySFTP = "sftp"
)

func (r DeployRepo) Normalized() DeployRepo {
	r.URL = strings.TrimSpace(r.URL)
	r.Ref = strings.TrimSpace(r.Ref)
	r.Token = strings.TrimSpace(r.Token)
	r.Method = strings.TrimSpace(strings.ToLower(r.Method))
	if r.URL != "" && r.Method == "" {
		r.Method = DeployGit
	}
	return r
}

func (r DeployRepo) Enabled() bool {
	r = r.Normalized()
	return r.URL != "" && r.Method == DeployGit
}

// MergeDeployRepo preserves token when the UI omits it on save.
func MergeDeployRepo(existing, incoming DeployRepo) DeployRepo {
	incoming = incoming.Normalized()
	existing = existing.Normalized()
	if incoming.Token == "" {
		incoming.Token = existing.Token
	}
	if incoming.URL == "" && incoming.Ref == "" && incoming.Method == "" {
		return existing
	}
	return incoming
}

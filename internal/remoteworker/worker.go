package remoteworker

import "goscan/internal/settings"

// Config is a normalized remote worker ready for SSH/API operations.
type Config struct {
	settings.RemoteWorker
	LocalVersion string
	AppRoot      string
	DeployRepo   settings.DeployRepo
}

func ConfigFrom(w settings.RemoteWorker, appRoot, localVersion string, repo settings.DeployRepo) Config {
	return Config{
		RemoteWorker: w.Normalized(),
		LocalVersion: localVersion,
		AppRoot:      appRoot,
		DeployRepo:   repo.Normalized(),
	}
}

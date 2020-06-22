package repo

import (
	"fmt"

	"github.com/bitnami-labs/chart-repository-syncer/api"
)

// ChartRepoAPI defines the methods a repo must implement.
type ChartRepoAPI interface {
	DownloadChart(filepath string, name string, version string, sourceRepo *api.Repo) error
	PublishChart(filepath string, targetRepo *api.Repo) error
	ChartExists(name string, version string, targetRepo *api.Repo) (bool, error)
}

// NewClient returns a client implementation for the given repo.
//
// The func is exposed as a var to allow tests to temporarily replace its
// implementation, e.g. to return a fake.
var NewClient = func(repo *api.Repo) (ChartRepoAPI, error) {
	switch repo.Kind {
	case api.Kind_HELM:
		return NewClassicHelmClient(repo), nil
	case api.Kind_CHARTMUSEUM:
		return NewChartMuseumClient(repo), nil
	default:
		return nil, fmt.Errorf("unsupported repo kind %q", repo.Kind)
	}
}

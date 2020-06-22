package repo

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"testing"

	"github.com/bitnami-labs/chart-repository-syncer/api"
	"github.com/bitnami-labs/chart-repository-syncer/pkg/chartmuseumtest"
)

var (
	sourceCM = &api.SourceRepo{
		Repo: &api.Repo{
			Url:  "http://fake.source.com",
			Kind: api.Kind_CHARTMUSEUM,
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
	}
	targetCM = &api.TargetRepo{
		Repo: &api.Repo{
			Url:  "http://fake.target.com",
			Kind: api.Kind_CHARTMUSEUM,
			Auth: &api.Auth{
				Username: "user",
				Password: "password",
			},
		},
		ContainerRegistry:   "test.registry.io",
		ContainerRepository: "test/repo",
	}
)

func TestPublishToChartmuseum(t *testing.T) {
	for _, test := range chartmuseumtest.Tests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update source repo url
			targetCM.Repo.Url = url

			// Create client for target repo
			tc, err := NewClient(targetCM.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}
			err = tc.PublishChart("../../testdata/apache-7.3.15.tgz", targetCM.Repo)
			if err != nil {
				t.Fatal(err)
			}

			// Check the chart really was added to the service's index.
			req, err := http.NewRequest("GET", targetCM.Repo.Url+"/api/charts/apache", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.SetBasicAuth(targetCM.Repo.Auth.Username, targetCM.Repo.Auth.Password)

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			charts := []*chartmuseumtest.ChartVersion{}
			if err := json.NewDecoder(resp.Body).Decode(&charts); err != nil {
				t.Fatal(err)
			}

			if got, want := len(charts), 1; got != want {
				t.Fatalf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Name, "apache"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
			if got, want := charts[0].Version, "7.3.15"; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestDownloadFromChartmuseum(t *testing.T) {
	for _, test := range chartmuseumtest.Tests {
		t.Run(test.Desc, func(t *testing.T) {
			// Check if the test should be skipped or allowed.
			test.Skip(t)

			url, cleanup := test.MakeServer(t)
			defer cleanup()

			// Update source repo url
			sourceCM.Repo.Url = url

			// Create client for source repo
			sc, err := NewClient(sourceCM.Repo)
			if err != nil {
				t.Fatal("could not create a client for the target repo", err)
			}

			// If testing real docker chartmuseum, we must push the chart before download it
			if test.Desc == "real service" {
				sc.PublishChart("../../testdata/apache-7.3.15.tgz", sourceCM.Repo)
			}

			// Create temporary working directory
			testTmpDir, err := ioutil.TempDir("", "c3tsyncer-tests")
			if err != nil {
				t.Errorf("Error creating temporary: %s", testTmpDir)
			}
			defer os.RemoveAll(testTmpDir)

			chartPath := path.Join(testTmpDir, "apache-7.3.15.tgz")
			err = sc.DownloadChart(chartPath, "apache", "7.3.15", sourceCM.Repo)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(chartPath); err != nil {
				t.Errorf("Expected %s to exists", chartPath)
			}
		})
	}
}

func TestChartExistsInChartMuseum(t *testing.T) {
	// Update source repo url
	// This repo is not a chartmuseum repo but there are no differences
	// for the ChartExists function.
	sourceCM.Repo.Url = "https://charts.bitnami.com/bitnami"
	// Create client for source repo
	sc, err := NewClient(sourceCM.Repo)
	if err != nil {
		t.Fatal("could not create a client for the source repo", err)
	}
	chartExists, err := sc.ChartExists("grafana", "1.5.2", sourceCM.Repo)
	if err != nil {
		t.Fatal(err)
	}
	if !chartExists {
		t.Errorf("grafana-1.5.2 chart should exists")
	}
}

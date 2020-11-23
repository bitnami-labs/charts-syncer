package helmclassic

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/juju/errors"
	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/charts-syncer/pkg/client/types"
)

func readErrorBody(r io.Reader) string {
	var s strings.Builder
	_, _ = io.Copy(&s, r)
	return s.String()
}

// Repo allows to operate a chart repository.
type Repo struct {
	url      *url.URL
	username string
	password string

	// NOTE: We need a lock for index to allow concurrency
	index *repo.IndexFile
}

// This allows test to replace the client index for testing.
var reloadIndex = func(r *Repo) error {
	u := r.GetIndexURL()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		klog.V(4).Infof("Using basic authentication %s:****", r.username)
		req.SetBasicAuth(r.username, r.password)
	}

	klog.V(4).Infof("GET %q", u)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotate(err, "fetching index.yaml")
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode == http.StatusNotFound {
		errorBody := readErrorBody(res.Body)
		return errors.Errorf("unable to fetch index.yaml, got HTTP Status: %s, Resp: %v", res.Status, errorBody)
	}

	// Create the index.yaml file to use the helm Go library, which does not
	// expose a Loader from bytes.
	f, err := ioutil.TempFile("", "index.*.yaml")
	if err != nil {
		return errors.Trace(err)
	}
	defer os.Remove(f.Name())

	// Write the body to file
	if _, err = io.Copy(f, res.Body); err != nil {
		return errors.Trace(err)
	}
	if err := f.Close(); err != nil {
		return errors.Trace(err)
	}

	index, err := repo.LoadIndexFile(f.Name())
	if err != nil {
		return errors.Trace(err)
	}

	r.index = index
	return nil
}

// New creates a Repo object from an api.Repo object.
func New(repo *api.Repo) (*Repo, error) {
	u, err := url.Parse(repo.GetUrl())
	if err != nil {
		return nil, errors.Trace(err)
	}

	return NewRaw(u, repo.GetAuth().GetUsername(), repo.GetAuth().GetPassword()), nil
}

// NewRaw creates a Repo object.
func NewRaw(u *url.URL, user string, pass string) *Repo {
	return &Repo{url: u, username: user, password: pass}
}

// GetDownloadURL returns the URL to download a chart
func (r *Repo) GetDownloadURL(n string, v string) (string, error) {
	chart, err := r.index.Get(n, v)
	if err != nil {
		return "", errors.Trace(err)
	}
	return chart.URLs[0], nil
}

// GetIndexURL returns the URL to download the index.yaml
func (r *Repo) GetIndexURL() string {
	u := *r.url
	u.Path = "/index.yaml"
	return u.String()
}

// List lists all chart names in a repo
func (r *Repo) List() ([]string, error) {
	if err := reloadIndex(r); err != nil {
		return []string{}, errors.Trace(err)
	}

	var names []string
	for name := range r.index.Entries {
		names = append(names, name)
	}

	return names, nil
}

// ListChartVersions lists all versions of a chart
func (r *Repo) ListChartVersions(name string) ([]string, error) {
	if err := reloadIndex(r); err != nil {
		return []string{}, errors.Trace(err)
	}

	cv, ok := r.index.Entries[name]
	if !ok {
		return []string{}, errors.Errorf("%q has no versions", name)
	}

	var versions []string
	for _, chart := range cv {
		versions = append(versions, chart.Version)
	}

	return versions, nil
}

// Fetch fetches a chart
func (r *Repo) Fetch(name string, version string, filename string) error {
	if err := reloadIndex(r); err != nil {
		return errors.Trace(err)
	}

	u, err := r.GetDownloadURL(name, version)
	if err != nil {
		return errors.Trace(err)
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return errors.Trace(err)
	}
	if r.username != "" && r.password != "" {
		klog.V(4).Infof("Using basic authentication %s:****", r.username)
		req.SetBasicAuth(r.username, r.password)
	}

	klog.V(4).Infof("GET %q", u)
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return errors.Annotatef(err, "fetching %s:%s chart", name, version)
	}
	defer res.Body.Close()

	// Check status code
	if res.StatusCode == http.StatusNotFound {
		errorBody := readErrorBody(res.Body)
		return errors.Errorf("unable to fetch %s:%s chart, got HTTP Status: %s, Resp: %v", name, version, res.Status, errorBody)
	}

	// Create the file
	f, err := os.Create(filename)
	if err != nil {
		return errors.Trace(err)
	}
	defer f.Close()

	// Write the body to file
	if _, err = io.Copy(f, res.Body); err != nil {
		return errors.Trace(err)
	}

	return nil
}

// Has checks if a repo has a specific chart
func (r *Repo) Has(name string, version string) (bool, error) {
	versions, err := r.ListChartVersions(name)
	if err != nil {
		return false, errors.Trace(err)
	}

	for _, v := range versions {
		if v == version {
			return true, nil
		}
	}
	return false, nil
}

// Upload uploads a chart to the repo
func (r *Repo) Upload(filepath string) error {
	klog.V(3).Infof("Publishing %q chart", filepath)
	return errors.Errorf("upload method is not supported yet")
}

// GetChartDetails returns the details of a chart
func (r *Repo) GetChartDetails(name string, version string) (*types.ChartDetails, error) {
	if err := reloadIndex(r); err != nil {
		return nil, errors.Trace(err)
	}

	cv, err := r.index.Get(name, version)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &types.ChartDetails{
		PublishedAt: cv.Created,
		Digest:      cv.Digest,
	}, nil
}

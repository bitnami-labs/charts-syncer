package main

import (
	"fmt"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/juju/errors"
	"github.com/mkmik/multierror"

	"github.com/bitnami-labs/charts-syncer/pkg/chart"
	"github.com/bitnami-labs/charts-syncer/pkg/config"
	"github.com/bitnami-labs/charts-syncer/pkg/helmcli"
	"github.com/bitnami-labs/charts-syncer/pkg/repo"
	"github.com/bitnami-labs/charts-syncer/pkg/utils"

	"github.com/spf13/cobra"
	"k8s.io/klog"
)

var (
	fromDate string // used for flags
)

func newSync() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Syncronize all the charts from a source repository to a target repository",
		Long:  `Syncronize all the charts from a source repository to a target repository`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return errors.Trace(sync())
		},
	}

	f := cmd.Flags()
	f.StringVar(&fromDate, "from-date", "", "Date you want to synchronize charts from. Format: YYYY-MM-DD")

	return cmd
}

func sync() error {
	var errs error
	// Load config file
	var syncConfig api.Config
	if err := config.Load(&syncConfig); err != nil {
		return errors.Trace(err)
	}
	if err := syncConfig.Validate(); err != nil {
		return errors.Trace(err)
	}
	source := syncConfig.Source
	target := syncConfig.Target

	// Create basic layout for date and parse flag to time type
	dateThreshold, err := utils.GetDateThreshold(fromDate)
	if err != nil {
		return errors.Trace(err)
	}
	// Load index.yaml info into index object
	sourceIndex, err := utils.LoadIndexFromRepo(source.Repo)
	if err != nil {
		return errors.Trace(fmt.Errorf("Error loading index.yaml: %w", err))
	}
	// Add target repo to helm CLI
	helmcli.AddRepoToHelm(target.Repo.Url, target.Repo.Auth)
	// Create client for target repo
	tc, err := repo.NewClient(target.Repo)
	if err != nil {
		return fmt.Errorf("could not create a client for the source repo: %w", err)
	}
	// Iterate over charts in source index
	for chartName := range sourceIndex.Entries {
		// Iterate over charts versions
		for i := range sourceIndex.Entries[chartName] {
			chartVersion := sourceIndex.Entries[chartName][i].Metadata.Version
			publishingTime := sourceIndex.Entries[chartName][i].Created
			if publishingTime.Before(dateThreshold) {
				continue
			}
			if chartExists, _ := tc.ChartExists(chartName, chartVersion, target.Repo); chartExists {
				continue
			}
			if dryRun {
				klog.Infof("dry-run: Chart %s-%s pending to be synced", chartName, chartVersion)
				continue
			}
			klog.Infof("Syncing %s-%s", chartName, chartVersion)
			if err := chart.Sync(chartName, chartVersion, source.Repo, target, sourceIndex, true); err != nil {
				errs = multierror.Append(errs, errors.Trace(err))
			}
		}
	}
	return errs
}

package chart

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/juju/errors"
	"k8s.io/klog"

	"github.com/bitnami-labs/charts-syncer/api"
)

// updateValuesFile performs some substitutions to a given values.yaml file.
func updateValuesFile(valuesFile string, targetRepo *api.TargetRepo) error {
	if err := updateContainerImageRegistry(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "Error updating %s file", valuesFile)
	}
	if err := updateContainerImageRepository(valuesFile, targetRepo); err != nil {
		return errors.Annotatef(err, "Error updating %s file", valuesFile)
	}
	return nil
}

// updateContainerImageRepository updates the container repository in a values.yaml file.
func updateContainerImageRepository(valuesFile string, targetRepo *api.TargetRepo) error {
	regex := regexp.MustCompile(`(?m)(repository:[[:blank:]])(.*)(/)`)
	values, err := ioutil.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := regex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], targetRepo.ContainerRepository, submatch[3])
		newContents := regex.ReplaceAllString(string(values), replaceLine)
		err = ioutil.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}

// updateContainerImageRegistry updates the container registry in a values.yaml file.
func updateContainerImageRegistry(valuesFile string, targetRepo *api.TargetRepo) error {
	regex := regexp.MustCompile(`(?m)(registry:[[:blank:]])(.*)(.*$)`)
	values, err := ioutil.ReadFile(valuesFile)
	if err != nil {
		return errors.Trace(err)
	}
	submatch := regex.FindStringSubmatch(string(values))
	if len(submatch) > 0 {
		replaceLine := fmt.Sprintf("%s%s%s", submatch[1], targetRepo.ContainerRegistry, submatch[3])
		newContents := regex.ReplaceAllString(string(values), replaceLine)
		err = ioutil.WriteFile(valuesFile, []byte(newContents), 0)
		if err != nil {
			return errors.Trace(err)
		}
	}
	return errors.Trace(err)
}

// updateReadmeFile performs some substitutions to a given README.md file.
func updateReadmeFile(readmeFile, sourceURL, targetURL, chartName, repoName string) error {
	klog.V(2).Infof("Updating README file")
	readme, err := ioutil.ReadFile(readmeFile)
	if err != nil {
		return errors.Trace(err)
	}
	// Update helm repo add with string replacement
	addBitnamiRepoLine := fmt.Sprintf("helm repo add bitnami %s", sourceURL)
	addCustomRepoLine := fmt.Sprintf("helm repo add %s %s", repoName, targetURL)
	newContent := strings.ReplaceAll(string(readme), addBitnamiRepoLine, addCustomRepoLine)
	// Update bitnami/chart references with regex
	regexString := fmt.Sprintf(`(?m)(\s)(bitnami/%s)(\s)`, chartName)
	regex := regexp.MustCompile(regexString)
	submatch := regex.FindStringSubmatch(string(readme))
	if len(submatch) > 0 {
		klog.V(2).Infof("Updating bitnami/ references")
		replaceText := fmt.Sprintf("%s%s/%s%s", submatch[1], repoName, chartName, submatch[3])
		newContent = regex.ReplaceAllString(newContent, replaceText)
	}
	err = ioutil.WriteFile(readmeFile, []byte(newContent), 0)
	if err != nil {
		return errors.Trace(err)
	}
	return nil
}

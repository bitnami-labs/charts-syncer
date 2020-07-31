package config

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/bitnami-labs/charts-syncer/api"
	"github.com/bitnami-labs/pbjson"
	"github.com/golang/protobuf/proto"
	"github.com/juju/errors"
	"github.com/spf13/viper"
	"k8s.io/klog"
	"sigs.k8s.io/yaml"
)

const (
	defaultRepoName = "myrepo"
)

// Load unmarshall config file into Config struct.
func Load(config *api.Config) error {
	err := yamlToProto(viper.ConfigFileUsed(), config)
	if err != nil {
		return errors.Trace(fmt.Errorf("error unmarshalling config file: %w", err))
	}
	if config.Target.RepoName == "" {
		klog.Warning("'target.repoName' property is empty. Using 'myrepo' default value")
		config.Target.RepoName = defaultRepoName
	}
	if config.Source.Repo.Auth == nil {
		sourceAuth := &api.Auth{}
		if su := os.Getenv("SOURCE_AUTH_USERNAME"); su != "" {
			sourceAuth.Username = su
		}
		if sp := os.Getenv("SOURCE_AUTH_PASSWORD"); sp != "" {
			sourceAuth.Password = sp
		}
		config.Source.Repo.Auth = sourceAuth
	}
	if config.Target.Repo.Auth == nil {
		targetAuth := &api.Auth{}
		if tu := os.Getenv("TARGET_AUTH_USERNAME"); tu != "" {
			targetAuth.Username = tu
		}
		if tp := os.Getenv("TARGET_AUTH_PASSWORD"); tp != "" {
			targetAuth.Password = tp
		}
		config.Target.Repo.Auth = targetAuth
	}
	return nil
}

// yamlToProto unmarshals `path` into the provided proto message
func yamlToProto(path string, v proto.Message) error {
	yamlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Trace(err)
	}
	jsonBytes, err := yaml.YAMLToJSONStrict(yamlBytes)
	if err != nil {
		return errors.Trace(err)
	}
	r := bytes.NewReader(jsonBytes)
	err = pbjson.NewDecoder(r).Decode(v)
	return errors.Trace(err)
}

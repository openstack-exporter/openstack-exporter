package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type CloudAuth struct {
	Username          string `yaml:"username"`
	Password          string `yaml:"password"`
	ProjectName       string `yaml:"project_name"`
	ProjectDomainName string `yaml:"project_domain_name"`
	UserDomainName    string `yaml:"user_domain_name"`
	AuthURL           string `yaml:"auth_url"`
}

type Cloud struct {
	Region             string    `yaml:"region_name"`
	IdentityAPIVersion string    `yaml:"identity_api_version"`
	IdentityInterface  string    `yaml:"identity_interface"`
	Auth               CloudAuth `yaml:"auth"`
}

type CloudConfig struct {
	Clouds map[string]Cloud `yaml:"clouds"`
}

func (config *CloudConfig) GetByName(name string) (*Cloud, error) {
	cloud, ok := config.Clouds[name]
	if !ok {
		return nil, fmt.Errorf("Cloud %s not found", name)
	}
	return &cloud, nil
}

func NewCloudConfigFromByteArray(data []byte) (*CloudConfig, error) {
	var config CloudConfig

	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, err
}

func NewCloudConfigFromFile(file string) (*CloudConfig, error) {
	var config CloudConfig

	data, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	return &config, err
}

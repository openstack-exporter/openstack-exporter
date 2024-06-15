package config

import (
	"fmt"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"net/url"
	"os"
)

const defaultConfigPath = "./config/config.yaml"

type ExporterConfig struct {
	Exporter *Exporter `yaml:"exporter"`
}

type Exporter struct {
	OSClientConfig string               `yaml:"os_client_config"`
	Port           int                  `yaml:"default_port"`
	MultiCloud     *MultiCloudConfig    `yaml:"multi_cloud"`
	Cloud          *CloudConfig         `yaml:"cloud"`
	Api            map[string]ApiConfig `yaml:"api"`
}

type MultiCloudConfig struct {
	IsEnabled   bool   `yaml:"is_enabled"`
	Description string `yaml:"description"`
}

type CloudConfig struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type ApiConfig struct {
	Metrics      *Metrics  `yaml:"metrics"`
	Prefix       *Prefix   `yaml:"prefix"`
	EndpointType *Endpoint `yaml:"endpoint_type"`
	CollectTime  *Collect  `yaml:"collect_time"`
	Services     []string  `yaml:"services"`
	Slow         *Skip     `yaml:"slow"`
	Deprecated   *Skip     `yaml:"deprecated"`
}

type Metrics struct {
	Uri         string `yaml:"uri"`
	Description string `yaml:"description"`
}

type Prefix struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type Endpoint struct {
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

type Collect struct {
	IsEnabled   *bool  `yaml:"is_enabled"`
	Description string `yaml:"description"`
}

type Skip struct {
	Skip        *bool  `yaml:"skip"`
	Description string `yaml:"description"`
}

// New creates a new ExporterConfig instance
func New(logger log.Logger) (*ExporterConfig, error) {
	path := os.Getenv("OS_EXPORTER_CONFIG_PATH")
	if path == "" {
		level.Warn(logger).Log("msg", "opentelekomcloud-exporter: warning: OS_EXPORTER_CONFIG_PATH is empty, trying to load from default path: ./config/config.yaml")
		path = defaultConfigPath
	}
	data, err := processFileOrURL(path, logger)
	if err != nil {
		level.Error(logger).Log("msg", "opentelekomcloud-exporter: error: error reading YAML file: %v", err)
	}
	var config ExporterConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		level.Error(logger).Log("msg", "opentelekomcloud-exporter: error: error unmarshalling YAML: %v", err)
	}
	return &config, nil
}

func processFileOrURL(input string, logger log.Logger) ([]byte, error) {
	var data []byte
	var err error

	if isURL(input) {
		data, err = downloadFile(input, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to download file: %w", err)
		}
	} else {
		data, err = os.ReadFile(input)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}
	}

	if isValidYAML(data) {
		return data, nil
	} else {
		return nil, fmt.Errorf("the file is not valid YAML")
	}
}

func isURL(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func downloadFile(fileURL string, logger log.Logger) ([]byte, error) {
	resp, err := http.Get(fileURL)
	if err != nil {
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			level.Error(logger).Log("msg", "opentelekomcloud-exporter: error: failed to close file: %v", err)
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download file: %s", resp.Status)
	}

	return io.ReadAll(resp.Body)
}

func isValidYAML(data []byte) bool {
	var yamlContent interface{}
	return yaml.Unmarshal(data, &yamlContent) == nil
}

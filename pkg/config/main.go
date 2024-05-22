package config

import (
	"fmt"
	"os"

	"github.com/go-yaml/yaml"
    "github.com/orangeAppsRu/custom-exporter/pkg/network"
    "github.com/orangeAppsRu/custom-exporter/pkg/proc"
)

const (
	lastRunReportPath = "/opt/puppetlabs/puppet/cache/state/last_run_report.yaml"
)

type Config struct {
    FileHashCollector struct {
        Enabled bool     `yaml:"enabled"`
        Files   []string `yaml:"files"`
    } `yaml:"fileHashCollector"`

    PortCollector struct {
        Enabled bool     `yaml:"enabled"`
        Targets []network.Target `yaml:"targets"`
    } `yaml:"portCollector"`

    ProcessCollector struct {
        Enabled bool     `yaml:"enabled"`
        Processes []proc.ProcessFilter `yaml:"processes"`
    } `yaml:"processCollector"`

    SystemCollector struct {
        Enabled bool `yaml:"enabled"`
    } `yaml:"systemCollector"`

    
    PuppetCollector struct {
        Enabled bool `yaml:"enabled"`
        LastRunReportPath string `yaml:"lastRunReportPath"`
    } `yaml:"puppetCollector"`
}


func ReadConfig(filePath string) (Config, error) {
    configFile, err := os.ReadFile(filePath)
    if err != nil {
        return Config{}, fmt.Errorf("error reading config file: %v", err)
    }

    var config Config
    if err := yaml.Unmarshal(configFile, &config); err != nil {
        return Config{}, fmt.Errorf("error parsing config file: %v", err)
    }
    // defaults
    if config.PuppetCollector.LastRunReportPath == "" {
        config.PuppetCollector.LastRunReportPath = lastRunReportPath
    }
    return config, nil
}

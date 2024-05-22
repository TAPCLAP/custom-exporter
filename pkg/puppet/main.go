package puppet


import (
	"fmt"
	"os"
	// "time"

	"github.com/go-yaml/yaml"
)


type lastRunReport struct {
	ConfigurationVersion int64  `yaml:"configuration_version"`
	TransactionCompleted bool   `yaml:"transaction_completed"`
}

type Puppet struct {
	lastRunReport *lastRunReport
}


func NewPuppet(lastRunReportPath string) (*Puppet) {
	l, err := parseYAMLFile(lastRunReportPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing YAML file: %v\n", err)
		l = &lastRunReport{
			ConfigurationVersion: 0,
			TransactionCompleted: false,
		}
	}


    return &Puppet{
		lastRunReport: l,
	}
}

func (p *Puppet) CheckCatalogLastCompile() int64 {
	return p.lastRunReport.ConfigurationVersion
}

func (p *Puppet) CheckCatalogLastCompileStatus() bool {
	return p.lastRunReport.TransactionCompleted
}

func parseYAMLFile(filePath string) (*lastRunReport, error) {
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %v", err)
	}

	var lastRunReport lastRunReport
	err = yaml.Unmarshal(yamlFile, &lastRunReport)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML data: %v", err)
	}
	return &lastRunReport, nil
}

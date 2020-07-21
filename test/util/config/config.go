package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kelseyhightower/envconfig"

	"gopkg.in/yaml.v2"
)

const (
	// ConfigPath path to config file
	ConfigPath = "config/config.yaml"
)

// Config type keeps general configuration
type Config struct {
	General struct {
		ReportDirAbsPath string `yaml:"report" envconfig:"REPORT_DIR_NAME"`
	} `yaml:"general"`
}

// NewConfig returs instacne Config type
func NewConfig() *Config {
	var c Config
	_, filename, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(filepath.Dir(filepath.Join(filepath.Dir(filename), "..")))
	confFile := filepath.Join(baseDir, ConfigPath)
	readFile(&c, confFile)
	c.General.ReportDirAbsPath = filepath.Join(baseDir, c.General.ReportDirAbsPath)
	readEnv(&c)
	return &c
}

func readFile(c *Config, cfgfile string) {
	f, err := os.Open(cfgfile)
	if err != nil {
		processError(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&c)
	if err != nil {
		processError(err)
	}
}

func readEnv(c *Config) {
	err := envconfig.Process("", c)
	if err != nil {
		processError(err)
	}
}

// GetReportPath returns full path to the report file
func (c *Config) GetReportPath(file string) string {
	reportFileName := strings.TrimSuffix(filepath.Base(file), filepath.Ext(filepath.Base(file)))
	return fmt.Sprintf("%s.xml", filepath.Join(c.General.ReportDirAbsPath, reportFileName))
}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

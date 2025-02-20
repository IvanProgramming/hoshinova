package config

import (
	"io/ioutil"
	"regexp"

	"github.com/HoloArchivists/hoshinova/logger"
	"gopkg.in/yaml.v2"
)

type Regexp regexp.Regexp

type Config struct {
	PollInterval int        `yaml:"poll_interval"`
	Workdir      string     `yaml:"workdir"`
	Logging      Logging    `yaml:"logging"`
	YTArchive    YTArchive  `yaml:"ytarchive"`
	Channels     []Channel  `yaml:"channels"`
	Uploaders    []Uploader `yaml:"uploaders"`
	Notifiers    []Notifier `yaml:"notifiers"`
}

type Logging struct {
	Level logger.LogLevel `yaml:"level"`
}

type YTArchive struct {
	Path    string   `yaml:"path"`
	Flags   []string `yaml:"flags"`
	Quality string   `yaml:"quality"`
}

type Channel struct {
	Name      string   `yaml:"name"`
	ChannelID string   `yaml:"id"`
	Filters   []Regexp `yaml:"filters"`
}

type Uploader struct {
	Name   string            `yaml:"name"`
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"config"`
}

type Notifier struct {
	Name   string            `yaml:"name"`
	Type   string            `yaml:"type"`
	Config map[string]string `yaml:"config"`
}

func LoadConfig() (*Config, error) {
	path := "config.yaml"
	var config Config
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *Regexp) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	re, err := regexp.Compile(s)
	if err != nil {
		return err
	}
	*r = Regexp(*re)
	return nil
}

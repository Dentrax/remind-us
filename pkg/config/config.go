/*
Copyright © 2021 Furkan Türkal

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

type Config struct {
	Integrations Integrations `yaml:"integrations"`
	Alerts       AlertConfig  `yaml:"alert"`
}

type Integrations struct {
	GitLab *GitLabIntegrationConfig `yaml:"gitlab"`
	RSS    *RSSIntegrationConfig    `yaml:"rss"`
}

type GitLabIntegrationConfig struct {
	Enabled string                  `yaml:"enabled"`
	Type    string                  `yaml:"type"`
	BaseURL string                  `yaml:"baseURL"`
	Token   string                  `yaml:"token"`
	Listen  IntegrationListenConfig `yaml:"listen"`
}

type RSSIntegrationConfig struct {
	Enabled string            `yaml:"enabled"`
	Sources []RSSSourceConfig `yaml:"sources"`
}

type RSSSourceConfig struct {
	URL        string         `yaml:"url"`
	Since      string         `yaml:"since"`
	MatchTitle RSSMatchConfig `yaml:"matchTitle"`
}

type RSSMatchConfig struct {
	Regexes  []string `yaml:"regexes"`
	Contains []string `yaml:"contains"`
}

type IntegrationListenConfig struct {
	Areas  []IntegrationAreaConfig
	Groups []int
}

type IntegrationAreaConfig struct {
	Type string `yaml:"type"`
}

type AlertConfig struct {
	Slack *SlackAlertConfig `yaml:"slack"`
}

type SlackAlertConfig struct {
	Enabled  string `yaml:"enabled"`
	Webhook  string `yaml:"webhook"`
	Channel  string `yaml:"channel"`
	Username string `yaml:"username"`
	Icon     string `yaml:"icon"`
}

func Load(path string) (*Config, error) {
	v := viper.New()

	v.SetTypeByDefaultValue(true)
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, errors.Wrapf(err, "unable to read config file: '%s'", path)
	}

	c := &Config{}

	err := v.Unmarshal(c)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal to Config struct")
	}

	return c, err
}

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

package slack

import (
	"errors"
	"github.com/Dentrax/remind-us/pkg/config"
	"github.com/slack-go/slack"
)

type Slack struct {
	config *config.SlackAlertConfig
	loaded bool
}

func (s *Slack) Name() string {
	return "Slack"
}

func (s *Slack) Load(config config.AlertConfig) error {
	s.config = &config.Slack
	s.loaded = true
	return nil
}

func (s *Slack) Alert(message interface{}) error {
	if !s.loaded {
		return errors.New("not loaded")
	}

	wh := message.(*slack.WebhookMessage)

	wh.Username = s.config.Username
	wh.Channel = s.config.Channel
	wh.IconEmoji = s.config.Icon

	return slack.PostWebhook(s.config.Webhook, message.(*slack.WebhookMessage))
}

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
	"strconv"

	"github.com/Dentrax/remind-us/pkg/config"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

var errAlert = errors.New("slack is not loaded")

type Slack struct {
	config *config.SlackAlertConfig
	loaded bool
}

func (s *Slack) Name() string {
	return "Slack"
}

func (s *Slack) Enabled(config config.AlertConfig) bool {
	if config.Slack == nil {
		return false
	}

	if config.Slack.Enabled == "" {
		return true
	}

	v, _ := strconv.ParseBool(config.Slack.Enabled)

	return v
}

func (s *Slack) Load(config config.AlertConfig) error {
	s.config = config.Slack
	s.loaded = true

	return nil
}

func (s *Slack) Alert(message interface{}) error {
	if !s.loaded {
		return errAlert
	}

	wh := message.(*slack.WebhookMessage) //nolint:forcetypeassert

	wh.Username = s.config.Username
	wh.Channel = s.config.Channel
	wh.IconEmoji = s.config.Icon

	err := slack.PostWebhook(s.config.Webhook, message.(*slack.WebhookMessage))
	if err != nil {
		return errors.Wrap(err, "unable to post webhook during alerting")
	}

	return nil
}

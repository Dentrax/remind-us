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

package integrations

import (
	"github.com/Dentrax/remind-us/pkg/config"
	"github.com/slack-go/slack"
)

type Integration struct {
	Validated bool
	Loaded    bool
}

type IIntegration interface {
	Name() string
	Enabled(config.Integrations) bool
	Validate(config.Integrations) error
	Load(config.Integrations) error
	GenerateSlackMessage(GenerateMessageOptions) (*slack.WebhookMessage, error)
}

type GenerateMessageOptions struct {
	// For integration name, i.e. Slack.
	For string
}

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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/Dentrax/remind-us/pkg/alerters"
	"github.com/Dentrax/remind-us/pkg/alerters/slack"
	"github.com/Dentrax/remind-us/pkg/config"
	"github.com/Dentrax/remind-us/pkg/integrations"
	"github.com/Dentrax/remind-us/pkg/integrations/gitlab"
	rss "github.com/Dentrax/remind-us/pkg/integrations/rss"
	"github.com/pkg/errors"
)

var (
	version     = "development"
	builtBy     = "compiler"
	commit      = "unknown"
	date        = time.Now().String()
	InitialTime = time.Now()
)

func main() {
	var configPath string

	flag.StringVar(&configPath, "config-file", "./config.yaml", "Configuration file path")
	v := flag.Bool("v", false, "Prints current version")
	flag.Parse()

	if *v {
		println(fmt.Sprintf(
			"remind-us %s (%s, %s, %s) on %s (%s)",
			version,
			builtBy,
			date,
			commit,
			runtime.GOOS,
			runtime.GOARCH,
		))
		os.Exit(0)
	}

	c, err := config.Load(configPath)
	if err != nil {
		log.Fatal(err)
	}

	err = Run(c)

	if err != nil {
		log.Fatal(err)
	}
}

func Run(config *config.Config) error {
	for _, i := range []integrations.IIntegration{
		&gitlab.GitLab{},
		&rss.RSS{
			InitialTime: InitialTime,
		},
	} {
		if !i.Enabled(config.Integrations) {
			continue
		}

		if err := i.Validate(config.Integrations); err != nil {
			return errors.Wrapf(err, "Could not validate '%s' config", i.Name())
		}

		err := i.Load(config.Integrations)
		if err != nil {
			return errors.Wrapf(err, "unable to load integration: '%s'", i.Name())
		}

		message, err := i.GenerateSlackMessage(integrations.GenerateMessageOptions{})
		if err != nil {
			return errors.Wrapf(err, "unable to generate slack message for integration: '%s'", i.Name())
		}

		if len(message.Attachments) == 0 {
			log.Printf("0 Attachments found for %s, no need to alert", i.Name())
			continue
		}

		for _, a := range []alerters.IAlerter{
			&slack.Slack{},
		} {
			if !a.Enabled(config.Alerts) {
				continue
			}

			err := a.Load(config.Alerts)
			if err != nil {
				return errors.Wrapf(err, "unable to load integration: '%s'", i.Name())
			}

			err = a.Alert(message)

			if err != nil {
				return errors.Wrapf(err, "unable to alert message for alerter: '%s', message: '%+v'", a.Name(), message)
			}

			log.Printf("%s alert success for channel: %s\n", a.Name(), message.Channel)
		}
	}

	return nil
}

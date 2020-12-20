<p align="center"><a href="https://github.com/Dentrax/remind-us" target="_blank"><img height="128" src="https://raw.githubusercontent.com/Dentrax/remind-us/master/.res/logo.png"></a></p>

<h1 align="center">remind-us</h1>

<div align="center">
    <strong>
    Schedule and generate custom reminders and send via custom alerters.
    </strong>
</div>

<br />

<p align="center">
  <a href="https://opensource.org/licenses/Apache-2.0"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg?style=flat-square" alt="Apache 2.0"></a>
  <a href="https://goreportcard.com/report/github.com/Dentrax/remind-us"><img src="https://goreportcard.com/badge/github.com/Dentrax/remind-us?style=flat-square" alt="Go Report"></a>
  <a href="https://github.com/Dentrax/remind-us/actions?workflow=test"><img src="https://img.shields.io/github/workflow/status/Dentrax/remind-us/Test?label=build&logo=github&style=flat-square" alt="Build Status"></a>
  <a href="https://github.com/Dentrax/remind-us/releases/latest"><img src="https://img.shields.io/github/release/Dentrax/remind-us.svg?style=flat-square" alt="GitHub release"></a>
</p>

<br />

*remind-us*, can generate custom _reminders_ and _alerters_ using a dynamically configured file. What you can do with this application is that how you can use it for. Deploy as a _Cron Job_, run automatically at start-up as a background process, etc.

**Warning:** A _PoC_ project, currently in *Alpha*.

## Features

> * NEW! Reminder: *GitLab (PRs)*
> * NEW! Alerter: *Slack (Webhook)*
> * Dynamic configuration support
> * Easy to use _integration_ and _alerter_ interfaces
> * Easy _cron job_ integration
> * ... and much more! - Explore and contribute!

## Screenshots

### GitLab: Slack

![Output](https://raw.githubusercontent.com/Dentrax/remind-us/master/.res/ss-gitlab-slack.png) 

## Installation

* Via Go
```bash
$ go get -u github.com/Dentrax/remind-us
```

* Via Docker
```bash
$ docker build -t remind-us \
                  --build-arg VERSION=`git describe --abbrev=0 --tag` \
                  --build-arg COMMIT=`git rev-parse --short HEAD` \
                  --build-arg DATE=`date -u +"%Y-%m-%dT%H:%M:%SZ"` \
                  -f Dockerfile .
```

## Usage

* Run the binary
```
$ remind-us --config-file "./config.yaml"
```

* Run on Docker
```
$ docker run -v `pwd`/config.yaml:/app/config.yaml -it remind-us
```

## Configuration

```yaml
integrations:
  gitlab:
    baseURL: <https://gitlab.com>
    token: <token>
    listen:
      areas:
        - type: "PR"
      groups:
        - <list-of-group-id>
alerts:
  slack:
    webhook: "<your-slack-webhook-endpoint>"
    channel: "<#channel>"
    username: "<username>"
    icon: "<:icon:>"
```

## TO-DO

* [ ] Add integration: [Jira](https://www.atlassian.com/software/jira)
* [ ] Add integration: [Todoist](https://todoist.com/)
* [ ] Add integration: [GitHub](https://github.com/)
* [ ] Add integration: *Quates*
* [ ] Add alerter: `stdout`
* [ ] Concurrency requests?

## License

*remind-us* was created by Furkan 'Dentrax' Türkal

The base project code is licensed under [Apache License 2.0](https://www.apache.org/licenses/LICENSE-2.0) unless otherwise specified. Please see the **[LICENSE](https://github.com/Dentrax/remind-us/blob/master/LICENSE)** file for more information.

<kbd>Best Regards</kbd>


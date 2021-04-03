package rss

import (
	"bufio"
	"os"
	"regexp"
	"testing"
	"time"

	"bou.ke/monkey"
	"github.com/Dentrax/remind-us/pkg/config"
	"github.com/Dentrax/remind-us/pkg/integrations"
	"github.com/mmcdole/gofeed"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func load(path, url string, matchConf config.RSSMatchConfig) (*RSS, error) {
	result := make(map[string]*gofeed.Feed)
	sourceConfigMap := make(map[string]config.RSSSourceConfig)
	sinceMap := make(map[string]time.Duration)
	matchTitleRegExpMap := make(map[string][]*regexp.Regexp)

	file, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read rss file from path: '%s'", path)
	}

	p := gofeed.NewParser()

	feed, err := p.Parse(bufio.NewReader(file))
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse rss file from path: '%s'", path)
	}

	result[url] = feed

	cfg := &config.RSSIntegrationConfig{
		Enabled: "true",
		Sources: []config.RSSSourceConfig{
			{
				URL:        url,
				Since:      "18h",
				MatchTitle: matchConf,
			},
		},
	}

	since, err := time.ParseDuration(cfg.Sources[0].Since)
	if err != nil {
		return nil, err
	}

	sinceMap[url] = since

	sourceConfigMap[url] = cfg.Sources[0]

	return &RSS{
		integrations.Integration{
			Validated: true,
			Loaded:    true,
		},
		result,
		sinceMap,
		sourceConfigMap,
		matchTitleRegExpMap,
		time.Now(),
		cfg,
	}, nil
}

func TestRSS_GenerateMessage_Slack(t *testing.T) {
	t.Parallel()

	patch := monkey.Patch(time.Now, func() time.Time { return time.Date(2021, time.March, 24, 20, 0o5, 7, 7, time.UTC) }) // Ref: https://stackoverflow.com/a/40639928/5685796

	t.Cleanup(func() {
		patch.Unpatch()
	})

	// Consider edit all date times manually at the `./testdata/integrations/rss/*.rss` path when adding a new RSS test
	tests := []struct {
		name    string
		path    string
		url     string
		config  config.RSSMatchConfig
		options integrations.GenerateMessageOptions
		want    slack.WebhookMessage
		wantErr bool
	}{
		{
			"it should parse HN FrontPage",
			"../../../testdata/integrations/rss/hn_frontpage.rss",
			"https://hnrss.org/frontpage",
			config.RSSMatchConfig{
				Contains: []string{"games"},
			},
			integrations.GenerateMessageOptions{},
			slack.WebhookMessage{
				Username:  "Username",
				IconEmoji: ":emoji:",
				IconURL:   "icon_url",
				Channel:   "#channel",
				Text:      "text",
				Attachments: []slack.Attachment{
					{
						Color:      "good",
						ID:         0,
						AuthorName: "Hacker News: Front Page",
						AuthorLink: "https://news.ycombinator.com/",
						Fields: []slack.AttachmentField{
							{
								Value: "• <https://news.ycombinator.com/item?id=26369653|Show HN: I made a note-taking app for roleplaying games like D&D> (56 minutes ago)",
							},
							{
								Value: "• <https://news.ycombinator.com/item?id=26369211|Dos.Zone – interactive database of DOS games> (1 hour ago)",
							},
						},
					},
				},
			},
			false,
		},
		{
			"it should parse r/golang",
			"../../../testdata/integrations/rss/r_golang_new.rss",
			"https://www.reddit.com/r/golang/new/.rss",
			config.RSSMatchConfig{
				Contains: []string{"book"},
			},
			integrations.GenerateMessageOptions{},
			slack.WebhookMessage{
				Username:  "Username",
				IconEmoji: ":emoji:",
				IconURL:   "icon_url",
				Channel:   "#channel",
				Text:      "text",
				Attachments: []slack.Attachment{
					{
						Color:      "good",
						ID:         0,
						AuthorName: "The Go Programming Language",
						AuthorLink: "https://www.reddit.com/r/golang/",
						Fields: []slack.AttachmentField{
							{
								Value: "• <https://www.reddit.com/r/golang/comments/mcey65/practical_go_lessons_book_700_pages_41_chapters/|Practical Go Lessons Book: 700+ pages, 41 chapters, 405+ drawings> (11 minutes ago)",
								Short: false,
							},
						},
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r, err := load(tt.path, tt.url, tt.config)
			assert.NoError(t, err)
			assert.NotNil(t, r)

			got, err := r.GenerateSlackMessage(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSlackMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NotNil(t, got)

			assert.Len(t, got.Attachments, len(tt.want.Attachments))

			for i, a := range tt.want.Attachments {
				assert.EqualValues(t, a, got.Attachments[i])
			}
		})
	}
}

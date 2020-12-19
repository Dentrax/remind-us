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

package gitlab

import (
	"bou.ke/monkey"
	"encoding/json"
	"fmt"
	"github.com/Dentrax/remind-us/pkg/config"
	"github.com/Dentrax/remind-us/pkg/integrations"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/xanzy/go-gitlab"
	"io/ioutil"
	"testing"
	"time"
)

func loadGitlab(rootPath string, groupPaths []string) (*GitLab, error) {
	groupScanResponses := make([]*GroupScanResponse, len(groupPaths))

	for i, gp := range groupPaths {
		groupFile, err := ioutil.ReadFile(gp)

		if err != nil {
			return nil, errors.Wrapf(err, "unable to read group file from path: '%s'", gp)
		}

		var groupProjects []*gitlab.Project

		err = json.Unmarshal(groupFile, &groupProjects)

		if err != nil {
			return nil, errors.Wrapf(err, "unable to unmarshal projects from path: '%s'", gp)
		}

		groupProjectScanResponses := make([]GroupProjectScanResponse, len(groupProjects))

		for j, pp := range groupProjects {
			pp := fmt.Sprintf("%s/projects_%d_merge_requests.json", rootPath, pp.ID)

			projectFile, err := ioutil.ReadFile(pp)

			if err != nil {
				return nil, errors.Wrapf(err, "unable to read group file from path: '%s'", gp)
			}

			var projectMRs []*gitlab.MergeRequest

			err = json.Unmarshal(projectFile, &projectMRs)

			if err != nil {
				return nil, errors.Wrapf(err, "unable to read project file from path: '%s'", pp)
			}

			groupProjectScanResponses[j] = GroupProjectScanResponse{
				Project: groupProjects[j],
				MRs:     projectMRs,
			}
		}

		groupScanResponses[i] = &GroupScanResponse{
			GroupID:  i,
			Projects: groupProjectScanResponses,
		}
	}

	return &GitLab{
		groupScanResponses,
		&config.GitLabIntegrationConfig{
			BaseURL: "http://gitlab.com",
		},
		true,
	}, nil
}

func TestGitLab_GenerateMessage(t *testing.T) {
	patch := monkey.Patch(time.Now, func() time.Time { return time.Date(2020, time.December, 13, 7, 7, 7, 7, time.UTC) }) //Ref: https://stackoverflow.com/a/40639928/5685796
	defer patch.Unpatch()

	tests := []struct {
		name       string
		rootPath   string
		groupPaths []string
		options    integrations.GenerateMessageOptions
		want       slack.WebhookMessage
		wantErr    bool
	}{
		{
			"it should parse empty groups projects array",
			"../../../testdata/integrations/gitlab",
			[]string{
				"../../../testdata/integrations/gitlab/groups_0_projects.json",
			},
			integrations.GenerateMessageOptions{},
			slack.WebhookMessage{
				Username:    "Username",
				IconEmoji:   ":emoji:",
				IconURL:     "icon_url",
				Channel:     "#channel",
				Text:        "text",
				Attachments: nil,
			},
			false,
		},
		{
			"it should parse single item of groups projects",
			"../../../testdata/integrations/gitlab",
			[]string{
				"../../../testdata/integrations/gitlab/groups_0_projects.json",
				"../../../testdata/integrations/gitlab/groups_1_projects.json",
			},
			integrations.GenerateMessageOptions{},
			slack.WebhookMessage{
				Username:    "Username",
				IconEmoji:   ":emoji:",
				IconURL:     "icon_url",
				Channel:     "#channel",
				Text:        "text",
				Attachments: nil,
			},
			false,
		},
		{
			"it should parse multi groups with multi projects and mrs",
			"../../../testdata/integrations/gitlab",
			[]string{
				"../../../testdata/integrations/gitlab/groups_2_projects.json",
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
						AuthorName: "baz 2",
						AuthorLink: "https://gitlab.com/foo/bar/baz.git",
						AuthorIcon: "https://gitlab.com/uploads/-/system/project/avatar/1162/envelope.png",
						Text: `There are <https://gitlab.com/foo/bar/baz/merge_requests?state=opened|3 open MRs> in <https://gitlab.com/foo/bar/baz|Foo / Bar / baz>. The oldest one is *8 weeks* old.

1 MR is reviewed and waiting:
✘ <https://gitlab.com/foo/bar/project/-/merge_requests/4|MR 4 - Title> (created *5 weeks* ago, updated 19 hours ago) by <@D4ntrax>

2 MRs are awaiting review:
✘ <https://gitlab.com/foo/bar/project/-/merge_requests/1|MR 1 - Title> (created *2 days* ago, updated 15 minutes ago) by <@D3ntrax>
✔ <https://gitlab.com/foo/bar/project/-/merge_requests/2|MR 2 - Title> (created *8 weeks* ago) by <@Dentrax>`,
						Footer:     "foo/bar",
						FooterIcon: "http://gitlab.com/uploads/-/system/group/avatar/229/bar.png",
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g, err := loadGitlab(tt.rootPath, tt.groupPaths)
			assert.NoError(t, err)
			assert.NotNil(t, g)

			got, err := g.GenerateSlackMessage(tt.options)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateMessage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.NotNil(t, got)

			assert.Len(t, got.Attachments, len(tt.want.Attachments))

			for i, a := range tt.want.Attachments {
				assert.Equal(t, a.Color, got.Attachments[i].Color)
				assert.Equal(t, a.AuthorName, got.Attachments[i].AuthorName)
				assert.Equal(t, a.AuthorLink, got.Attachments[i].AuthorLink)
				assert.Equal(t, a.AuthorIcon, got.Attachments[i].AuthorIcon)
				assert.Equal(t, a.Title, got.Attachments[i].Title)
				assert.Equal(t, a.Text, got.Attachments[i].Text)
				assert.Equal(t, a.Footer, got.Attachments[i].Footer)
				assert.Equal(t, a.FooterIcon, got.Attachments[i].FooterIcon)
				assert.NotEmpty(t, got.Attachments[i].Ts)
			}
		})
	}
}

func Benchmark_GenerateMessage(b *testing.B) {
	g, _ := loadGitlab("../../../testdata/integrations/gitlab", []string{
		"../../../testdata/integrations/gitlab/groups_0_projects.json",
		"../../../testdata/integrations/gitlab/groups_1_projects.json",
		"../../../testdata/integrations/gitlab/groups_2_projects.json",
	})

	o := integrations.GenerateMessageOptions{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = g.GenerateSlackMessage(o)
	}
}

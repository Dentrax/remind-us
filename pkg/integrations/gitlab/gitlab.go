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
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/Dentrax/remind-us/pkg/config"
	"github.com/Dentrax/remind-us/pkg/integrations"
	"github.com/hako/durafmt"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/xanzy/go-gitlab"
	"strconv"
	"strings"
	"time"
)

type GitLab struct {
	Result *[]GroupScanResponse
	config *config.GitLabIntegrationConfig
}

type GroupScanResponse struct {
	GroupID  uint16
	Projects []GroupProjectScanResponse
}

type GroupProjectScanResponse struct {
	Project *gitlab.Project
	MRs     []*gitlab.MergeRequest
}

func (g GitLab) Name() string {
	return "GitLab"
}

func (g GitLab) Load(config config.IntegrationConfig) error {
	git, err := gitlab.NewClient(config.GitLab.Token, gitlab.WithBaseURL(config.GitLab.BaseURL))

	if err != nil {
		return errors.Wrap(err, "Unable to generate GitLab Client")
	}

	result := make([]GroupScanResponse, len(config.GitLab.Listen.Groups))

	for i, g := range config.GitLab.Listen.Groups {
		projects, _, err := git.Groups.ListGroupProjects(g, &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 100,
			},
		})

		if err != nil {
			return errors.Wrapf(err, "Unable to list projects for group id: %d", g)
		}

		result[i] = GroupScanResponse{
			GroupID:  g,
			Projects: make([]GroupProjectScanResponse, len(projects)),
		}

		stateType := "opened"

		for j, p := range projects {
			mrs, _, err := git.MergeRequests.ListProjectMergeRequests(p.ID,
				&gitlab.ListProjectMergeRequestsOptions{
					ListOptions: gitlab.ListOptions{
						Page:    1,
						PerPage: 100,
					},
					State: &stateType,
				},
			)

			if err != nil {
				return errors.Wrapf(err, "Unable to list merge requests for project id: %d, group id: %d", p.ID, g)
			}

			result[i].Projects[j] = GroupProjectScanResponse{
				Project: p,
				MRs:     mrs,
			}
		}
	}

	g.Result = &result
	g.config = &config.GitLab

	return nil
}

func (g GitLab) GenerateMessage(options integrations.GenerateMessageOptions) (slack.WebhookMessage, error) {
	var attachments []slack.Attachment

	for _, r := range *g.Result {
		for _, p := range r.Projects {
			var resultProject bytes.Buffer

			oldest := time.Now()
			openMRs := 0
			for _, m := range p.MRs {
				if strings.EqualFold(m.State, "opened") {
					openMRs++
					if m.CreatedAt != nil && m.CreatedAt.Before(oldest) {
						oldest = *m.CreatedAt
					}
				}
			}

			if openMRs <= 0 {
				continue
			}

			resultProject.WriteString(fmt.Sprintf("There are <%s|%d open MRs> in <%s|%s>. The oldest one is %s old.", fmt.Sprintf("%s/merge_requests?state=opened", p.Project.WebURL), openMRs, p.Project.NameWithNamespace, p.Project.WebURL, durafmt.Parse(time.Since(oldest)).LimitFirstN(1).String()))
			resultProject.WriteString("\n")

			var reviewedMRs []*gitlab.MergeRequest
			var awaitingMRs []*gitlab.MergeRequest
			for _, m := range p.MRs {
				if strings.EqualFold(m.State, "opened") {
					if m.Upvotes > 0 || m.Downvotes > 0 || m.UserNotesCount > 0 {
						reviewedMRs = append(reviewedMRs, m)
					} else {
						awaitingMRs = append(awaitingMRs, m)
					}
				}
			}

			GetDateInfo := func(created, updated *time.Time) string {
				GetTimeText := func(t *time.Time) string {
					d := durafmt.Parse(time.Since(*t)).LimitFirstN(1)
					if d.Duration().Hours() >= 48 {
						return fmt.Sprintf("*%s*", d.String())
					}
					return d.String()
				}

				if created != nil && updated != nil {
					if created.Equal(*updated) {
						return fmt.Sprintf("(created %s ago)", GetTimeText(created))
					}
					return fmt.Sprintf("(created %s ago, updated %s ago)", GetTimeText(created), GetTimeText(updated))
				} else if created != nil {
					return fmt.Sprintf("(created %s ago)", GetTimeText(created))
				}
				return ""
			}

			AppendMRInfo := func(buffer *bytes.Buffer, m *gitlab.MergeRequest) {
				GetCanBeMerged := func(b string) rune {
					if strings.EqualFold(b, "can_be_merged") {
						return '✔'
					}
					return '✘'
				}(m.MergeStatus)
				buffer.WriteString(fmt.Sprintf("\n%c <%s|%s> %s", GetCanBeMerged, m.WebURL, m.Title, GetDateInfo(m.CreatedAt, m.UpdatedAt)))
				buffer.WriteString(fmt.Sprintf(" by <@%s>", m.Author.Username))
			}

			if len(reviewedMRs) > 1 {
				resultProject.WriteString(fmt.Sprintf("\n%d MRs are reviewed and waiting:", len(reviewedMRs)))
			} else if len(reviewedMRs) == 1 {
				resultProject.WriteString("\n1 MR is reviewed and waiting:")
			}

			for _, m := range reviewedMRs {
				AppendMRInfo(&resultProject, m)
			}

			resultProject.WriteString("\n")

			if len(awaitingMRs) > 1 {
				resultProject.WriteString(fmt.Sprintf("\n%d MRs are awaiting review:", len(awaitingMRs)))
			} else if len(awaitingMRs) == 1 {
				resultProject.WriteString("\n1 MR is awaiting review:")
			}

			for _, m := range awaitingMRs {
				AppendMRInfo(&resultProject, m)
			}

			attachments = append(attachments, slack.Attachment{
				Color:      "good",
				AuthorName: p.Project.Name,
				AuthorLink: p.Project.HTTPURLToRepo,
				AuthorIcon: p.Project.AvatarURL,
				Title:      "title",
				Text:       resultProject.String(),
				Footer:     p.Project.Namespace.FullPath,
				FooterIcon: fmt.Sprintf("%s/%s", g.config.BaseURL, p.Project.Namespace.AvatarURL),
				Ts:         json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
			})
		}
	}

	return slack.WebhookMessage{
		Attachments: attachments,
		Username:    "Username",
		IconEmoji:   ":emoji:",
		IconURL:     "icon_url",
		Channel:     "#channel",
		Text:        "text",
	}, nil
}

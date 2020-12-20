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
	"log"
	"strconv"
	"strings"
	"time"
)

type GitLab struct {
	Result []*GroupScanResponse
	config *config.GitLabIntegrationConfig
	loaded bool
}

type GroupScanResponse struct {
	GroupID  int
	Projects []GroupProjectScanResponse
}

type GroupProjectScanResponse struct {
	Project *gitlab.Project
	MRs     []*gitlab.MergeRequest
}

func (g *GitLab) Name() string {
	return "GitLab"
}

func (g *GitLab) Load(config config.IntegrationConfig) error {
	git, err := gitlab.NewClient(config.GitLab.Token, gitlab.WithBaseURL(config.GitLab.BaseURL))

	if err != nil {
		return errors.Wrap(err, "Unable to generate GitLab Client")
	}

	g.Result = make([]*GroupScanResponse, len(config.GitLab.Listen.Groups))

	for i, l := range config.GitLab.Listen.Groups {
		projects, _, err := git.Groups.ListGroupProjects(l, &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    1,
				PerPage: 100,
			},
		})

		if err != nil {
			return errors.Wrapf(err, "Unable to list projects for group id: '%d'", l)
		}

		log.Printf("%d project(s) found in group %d\n", len(projects), l)

		g.Result[i] = &GroupScanResponse{
			GroupID:  l,
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
				return errors.Wrapf(err, "Unable to list merge requests for project id: %d, group id: %d", p.ID, l)
			}

			log.Printf("%d MR(s) found in project %s\n", len(mrs), p.Name)

			g.Result[i].Projects[j] = GroupProjectScanResponse{
				Project: p,
				MRs:     mrs,
			}
		}
	}

	g.config = &config.GitLab
	g.loaded = true

	return nil
}

func (g *GitLab) GenerateSlackMessage(options integrations.GenerateMessageOptions) (*slack.WebhookMessage, error) {
	if !g.loaded {
		return nil, errors.New("not loaded")
	}

	var attachments []slack.Attachment

	for _, r := range g.Result {
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

			GetTimeText := func(t *time.Time) string {
				d := durafmt.Parse(time.Since(*t)).LimitFirstN(1)
				if d.Duration().Hours() >= 48 {
					return fmt.Sprintf("*%s*", d.String())
				}
				return d.String()
			}

			GetMRKeyword := func(link string, openMRs int) string {
				if openMRs > 1 {
					return fmt.Sprintf("are <%s|%d open MRs>", link, openMRs)
				}
				return fmt.Sprintf("is <%s|%d open MR>", link, openMRs)
			}(fmt.Sprintf("%s/merge_requests?state=opened", p.Project.WebURL), openMRs)

			resultProject.WriteString(fmt.Sprintf("There %s in <%s|%s>.", GetMRKeyword, p.Project.WebURL, p.Project.NameWithNamespace))

			if openMRs > 1 {
				resultProject.WriteString(fmt.Sprintf(" The oldest one is %s old.", GetTimeText(&oldest)))
			}

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
						return '✓'
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

			if len(reviewedMRs) >= 1 {
				resultProject.WriteString("\n")
			}

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
				Text:       resultProject.String(),
				Footer:     p.Project.Namespace.FullPath,
				FooterIcon: fmt.Sprintf("%s%s", g.config.BaseURL, p.Project.Namespace.AvatarURL),
				Ts:         json.Number(strconv.FormatInt(time.Now().Unix(), 10)),
			})
		}
	}

	return &slack.WebhookMessage{
		Attachments: attachments,
	}, nil
}

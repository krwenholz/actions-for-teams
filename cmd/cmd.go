package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	github "github.com/shurcooL/githubv4"
)

type Issue struct {
	Number    github.Int
	ID        github.ID
	Assignees struct {
		Nodes []Assignee
	} `graphql:"assignees(first:10)"`
	Labels struct {
		Nodes []Label
	} `graphql:"labels(first:10)"`
	ProjectsV2 struct {
		Nodes []Project
	} `graphql:"projectsV2(first:10)"`
}

type Assignee struct {
	Login string
}

type Label struct {
	Name string
}

type Project struct {
	Number github.Int
}

type issueQuery struct {
	Repository struct {
		Issue Issue `graphql:"issue(number: $number)"`
	} `graphql:"repository(owner: $organization, name: $repository)"`
}

func (cc *Commands) addToProjectCmd() *cobra.Command {
	var (
		issues []int64
	)

	c := &cobra.Command{
		Use:   "add-to-project",
		Short: "Based on data passed and our config, add an issue or issues to GitHub project(s).",
	}
	c.Flags().Int64SliceVar(&issues, "issues", []int64{}, "The list of issue ids to update project memberships for.")

	c.RunE = func(c *cobra.Command, _ []string) error {
		ctx := c.Root().Context()
		cc.initGHClient()
		cc.initTeams()

		for _, id := range issues {
			fmt.Println("Updating project memberships for issue", id)

			var q issueQuery
			qArgs := map[string]interface{}{
				"organization": github.String(cc.RepOwner),
				"repository":   github.String(cc.Repository),
				"number":       github.Int(id),
			}
			err := cc.GHClient.Query(ctx, &q, qArgs)
			if err != nil {
				return err
			}

			fmt.Printf("Issue fetched %d with assignees %v and labels %v\n", id, q.Repository.Issue.Assignees.Nodes, q.Repository.Issue.Labels.Nodes)

			for _, team := range cc.Teams.Teams {
				if !hasTeamAssignee(team, &q.Repository.Issue) && !hasTeamLabel(team, &q.Repository.Issue) {
					continue
				}
				var pq struct {
					Organization struct {
						ProjectV2 struct {
							ID github.ID
						} `graphql:"projectV2(number: $number)"`
					} `graphql:"organization(login: $organization)"`
				}
				pqArgs := map[string]interface{}{
					"organization": github.String(cc.RepoOwner),
					"number":       github.Int(team.GitHubPrivateBoardID),
				}
				err := cc.GHClient.Query(ctx, &pq, pqArgs)
				if err != nil {
					return err
				}

				var m struct {
					AddProjectV2ItemById struct {
						Item struct {
							ID github.ID
						}
					} `graphql:"addProjectV2ItemById(input: $input)"`
				}
				input := github.AddProjectV2ItemByIdInput{
					ContentID: q.Repository.Issue.ID,
					ProjectID: pq.Organization.ProjectV2.ID,
				}
				err = cc.GHClient.Mutate(ctx, &m, input, nil)
				if err != nil {
					return err
				}

				fmt.Printf("Added issue %d to %s's board %d\n", id, team.Name, team.GitHubPrivateBoardID)
			}
		}
		return nil
	}

	return c
}

func hasTeamAssignee(t Team, i *Issue) bool {
	for _, member := range t.Members {
		for _, assignee := range i.Assignees.Nodes {
			if member.GitHubAlias == assignee.Login {
				return true
			}
		}
	}
	return false
}

func hasTeamLabel(t Team, i *Issue) bool {
	for _, tl := range t.GitHubLabels {
		for _, il := range i.Labels.Nodes {
			if tl == il.Name {
				return true
			}
		}
	}
	return false
}

func isInProject(i *Issue, p int64) bool {
	for _, ip := range i.ProjectsV2.Nodes {
		if ip.Number == github.Int(p) {
			return true
		}
	}
	return false
}

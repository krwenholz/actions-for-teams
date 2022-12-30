package cmd

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/bradleyfalzon/ghinstallation/v2"
	github "github.com/shurcool/githubv4"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type Member struct {
	GitHubAlias string
}

type Team struct {
	GitHubLabels         []string
	GitHubPrivateBoardID int64
	GitHubPrivateName    string
	Members              []Member
	Name                 string
}

type TeamsInput struct {
	Members []Member
	Teams   []Team
}

type Commands struct {
	RepoOwner        string
	Repository       string
	GHClientID       int64
	GHInstallationID int64
	GHPem            string // ./secrets/local/github-app.private-key.pem
	GHClient         github.Client
	TeamsPath        string
	Teams            TeamsInput

	//teams, err := cfgprog.Driver().Teams(ctx)
	//if err != nil {
	//	return err
	//}
}

// rootCmd represents the base command when called without any subcommands
func (cc *Commands) Root() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "actions-for-teams",
		Short: "A small suite of automations for teams, intended for use on GitHub Actions",
	}
	rootCmd.PersistentFlags().StringVar(&cc.RepoOwner, "owner", "", "Repository owner in GitHub.")
	rootCmd.PersistentFlags().StringVar(&cc.Repository, "repository", "", "Repository name in GitHub.")
	rootCmd.PersistentFlags().Int64Var(&cc.GHClientID, "client-id", 0, "Client ID for the GitHub App to use.")
	rootCmd.PersistentFlags().Int64Var(&cc.GHInstallationID, "installation-id", 0, "Installation ID for the GitHub App to use.")
	rootCmd.PersistentFlags().StringVar(&cc.GHPem, "pem", "", "Path to pem file for GitHub App to use.")
	rootCmd.PersistentFlags().StringVar(&cc.TeamsPath, "teams", "", "Path to team definition file.")

	//set variables and add commands
	rootCmd.AddCommand(
		cc.addToProjectCmd(),
	)

	return rootCmd
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	var cc Commands
	err := cc.Root().Execute()
	if err != nil {
		os.Exit(1)
	}
}

func (cc *Commands) initGHClient() error {
	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, cc.GHClientID, cc.GHInstallationID, cc.GHPem)
	if err != nil {
		return err
	}

	cc.GHClient = *github.NewClient(&http.Client{Transport: itr})
	return nil
}

func (cc *Commands) initTeams() error {
	f, err := os.Open(cc.TeamsPath)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal([]byte(b), &cc.Teams)
	if err != nil {
		return err
	}

	return nil
}

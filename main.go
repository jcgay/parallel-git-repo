package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	app := cli.NewApp()
	app.Name = "Parallel Git Repositories"
	app.Usage = "Execute commands on multiple Git repositories at the same time!"
	app.Commands = buildCommands()
	app.Run(os.Args)
}

type Repositories interface {
	List() []string
}

type RegisteredRepositories struct{}

func (repos *RegisteredRepositories) List() []string {
	return []string{"/Users/jcgay/dev/maven-notifier", "/Users/jcgay/dev/maven-color"}
}

type RunnableCommand interface {
	Executable() string
	Options() []string
}

type GitPullCommand struct{}

func (command *GitPullCommand) Executable() string {
	return "git"
}

func (command *GitPullCommand) Options() []string {
	return []string{"pull", "--rebase=preserve"}
}

type GitShowCurrentBranchCommand struct{}

func (command *GitShowCurrentBranchCommand) Executable() string {
	return "git"
}

func (command *GitShowCurrentBranchCommand) Options() []string {
	return []string{"symbolic-ref", "--short", "HEAD"}
}

type EchoCommand struct{}

func (command *EchoCommand) Executable() string {
	return "pwd"
}

func (command *EchoCommand) Options() []string {
	return []string{}
}

type Runner struct {
	RunnableCommand

	repos  Repositories
	writer io.Writer
}

func NewRunner(command RunnableCommand) *Runner {
	return &Runner{
		RunnableCommand: command,
		repos:           &RegisteredRepositories{},
		writer:          os.Stdout,
	}
}

func (runner *Runner) Run() {
	for _, repo := range runner.repos.List() {
		output := new(bytes.Buffer)
		errorOutput := new(bytes.Buffer)

		command := exec.Command(runner.RunnableCommand.Executable(), runner.RunnableCommand.Options()...)
		command.Stdout = output
		command.Stderr = errorOutput
		command.Dir = repo
		command.Run()

		fmt.Fprint(runner.writer, filepath.Base(repo)+": "+output.String())
	}
}

func buildCommands() []cli.Command {
	commands := make([]cli.Command, 1)
	commands = append(commands, cli.Command{Name: "echo", Action: func(context *cli.Context) {
		NewRunner(&EchoCommand{}).Run()
	},
	}, cli.Command{Name: "pull", Action: func(context *cli.Context) {
		NewRunner(&GitPullCommand{}).Run()
	}}, cli.Command{Name: "current-branch", Action: func(context *cli.Context) {
		NewRunner(&GitShowCurrentBranchCommand{}).Run()
	}})
	return commands
}

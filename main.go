package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
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
	Output(output string) string
}

type GitPullCommand struct{}

func (command *GitPullCommand) Executable() string {
	return "git"
}

func (command *GitPullCommand) Options() []string {
	return []string{"pull", "--rebase=preserve"}
}

func (command *GitPullCommand) Output(output string) string {
	return output
}

type GitShowCurrentBranchCommand struct{}

func (command *GitShowCurrentBranchCommand) Executable() string {
	return "git"
}

func (command *GitShowCurrentBranchCommand) Options() []string {
	return []string{"symbolic-ref", "--short", "HEAD"}
}

func (command *GitShowCurrentBranchCommand) Output(output string) string {
	return output
}

type EchoCommand struct{}

func (command *EchoCommand) Executable() string {
	return "pwd"
}

func (command *EchoCommand) Options() []string {
	return []string{}
}

func (command *EchoCommand) Output(output string) string {
	return output
}

type GitMergeCommand struct{}

func (command *GitMergeCommand) Executable() string {
	return "git"
}

func (command *GitMergeCommand) Options() []string {
	return []string{"merge", "$1"}
}

func (command *GitMergeCommand) Output(output string) string {
	return output
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

func (runner *Runner) Run(args cli.Args) {
	for _, repo := range runner.repos.List() {
		output := new(bytes.Buffer)
		errorOutput := new(bytes.Buffer)

		command := exec.Command(runner.RunnableCommand.Executable(), forwardArgs(runner.RunnableCommand.Options(), args)...)
		command.Stdout = output
		command.Stderr = errorOutput
		command.Dir = repo
		err := command.Run()

		if err != nil {
			fmt.Fprint(runner.writer, filepath.Base(repo)+": "+err.Error()+"\n"+errorOutput.String())
		} else {
			fmt.Fprint(runner.writer, filepath.Base(repo)+": "+runner.RunnableCommand.Output(output.String()))
		}
	}
}

var option = regexp.MustCompile(`\$([0-9]+)`)

func forwardArgs(opts []string, args cli.Args) []string {
	result := make([]string, 0)
	for _, opt := range opts {
		if opt == "$@" {
			result = append(result, args...)
		} else if option.MatchString(opt) {
			result = append(result, option.ReplaceAllStringFunc(opt, func(substring string) string {
				index, _ := strconv.Atoi(substring[1:])
				return args[index-1]
			}))
		} else {
			result = append(result, opt)
		}
	}
	return result
}

func buildCommands() []cli.Command {
	commands := make([]cli.Command, 1)
	commands = append(commands, cli.Command{Name: "echo", Action: func(context *cli.Context) {
		NewRunner(&EchoCommand{}).Run(context.Args())
	},
	}, cli.Command{Name: "pull", Action: func(context *cli.Context) {
		NewRunner(&GitPullCommand{}).Run(context.Args())
	}}, cli.Command{Name: "current-branch", Action: func(context *cli.Context) {
		NewRunner(&GitShowCurrentBranchCommand{}).Run(context.Args())
	}})
	return commands
}

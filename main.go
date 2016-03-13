package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/mitchellh/go-homedir"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
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

type RegisteredRepositories struct {
	homeDir string
}

func (repos *RegisteredRepositories) List() []string {
	data, err := ioutil.ReadFile(repos.homeDir + "/.parallel-git-repositories")
	if err != nil {
		log.Fatalf("Can't read configuration file %s/.parallel-git-repositories, verify that the file is correctly set...\n%v", repos.homeDir, err)
	}

	config := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Configuration file %s/.parallel-git-repositories is not a valid yaml file.\n%v", repos.homeDir, err)
	}

	all := config["repositories"].([]interface{})
	result := make([]string, len(all))
	for i, repo := range all {
		result[i] = repo.(string)
	}
	return result
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
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalf("Cannot read user home directory.\n%v", err)
	}
	return &Runner{
		RunnableCommand: command,
		repos:           &RegisteredRepositories{home},
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

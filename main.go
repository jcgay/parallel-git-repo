package main

import (
	"bytes"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/jcgay/parallel-git-repo/command"
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
	"strings"
	"sync"
)

var VERSION = "unknown-snapshot"

func main() {
	app := cli.NewApp()
	app.Name = "Parallel Git Repositories"
	app.Usage = "Execute commands on multiple Git repositories in parallel!"
	app.Commands = buildCommands()
	app.Version = VERSION
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
	Output(output string, errOutput string, err error) string
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
	wg := sync.WaitGroup{}
	for _, repo := range runner.repos.List() {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			output := new(bytes.Buffer)
			errorOutput := new(bytes.Buffer)

			command := exec.Command(runner.RunnableCommand.Executable(), forwardArgs(runner.RunnableCommand.Options(), args)...)
			command.Stdout = output
			command.Stderr = errorOutput
			command.Dir = repo
			err := command.Run()

			fmt.Fprintln(runner.writer, filepath.Base(repo)+": "+runner.RunnableCommand.Output(strings.TrimSpace(output.String()), strings.TrimSpace(errorOutput.String()), err))
		}(repo)
	}
	wg.Wait()
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
		NewRunner(&command.Echo{}).Run(context.Args())
	},
	}, cli.Command{Name: "pull", Usage: "Fetch from and integrate with another repository or a local branch", Action: func(context *cli.Context) {
		NewRunner(&command.GitPull{}).Run(context.Args())
	}}, cli.Command{Name: "current-branch", Usage: "Get current checkout branch from a repository", Action: func(context *cli.Context) {
		NewRunner(&command.GitShowCurrentBranch{}).Run(context.Args())
	}}, cli.Command{Name: "merge", Usage: "Join two or more development histories together", Action: func(context *cli.Context) {
		NewRunner(&command.GitMerge{}).Run(context.Args())
	}}, cli.Command{Name: "fetch", Usage: "Download objects and refs from another repository and prune", Action: func(context *cli.Context) {
		NewRunner(&command.GitFetch{}).Run(context.Args())
	}})
	return commands
}

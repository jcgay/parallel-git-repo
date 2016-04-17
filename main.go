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
	ListRepositories() []string
}

type Commands interface {
	ListCommands() map[string]string
}

type Configuration struct {
	content map[interface{}]interface{}
}

func NewConfiguration(homeDir string) *Configuration {
	data, err := ioutil.ReadFile(homeDir + "/.parallel-git-repositories")
	if err != nil {
		log.Fatalf("Can't read configuration file %s/.parallel-git-repositories, verify that the file is correctly set...\n%v", homeDir, err)
	}

	config := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatalf("Configuration file %s/.parallel-git-repositories is not a valid yaml file.\n%v", homeDir, err)
	}

	return &Configuration{config}
}

func (config *Configuration) ListRepositories() []string {
	all := config.content["repositories"].([]interface{})
	result := make([]string, len(all))
	for i, repo := range all {
		result[i] = repo.(string)
	}
	return result
}

func (config *Configuration) ListCommands() map[string]string {
	all := config.content["commands"].(map[interface{}]interface{})
	result := make(map[string]string)
	for key, value := range all {
		result[key.(string)] = value.(string)
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

func NewRunner(command RunnableCommand, repos Repositories) *Runner {
	return &Runner{
		RunnableCommand: command,
		repos:           repos,
		writer:          os.Stdout,
	}
}

func (runner *Runner) Run(args cli.Args) {
	wg := sync.WaitGroup{}
	for _, repo := range runner.repos.ListRepositories() {
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
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalf("Cannot read user home directory.\n%v", err)
	}
	configuration := NewConfiguration(home)

	quietFlag := []cli.Flag{
		cli.BoolFlag{
			Name:  "quiet, q",
			Usage: "Do not print stdout commands result, only stderr will be shown",
		},
	}

	commands := make([]cli.Command, 0)
	commands = append(commands, cli.Command{Name: "run", Usage: "Run an arbitrary command", Flags: quietFlag, Action: func(context *cli.Context) {
		NewRunner(&command.Run{ToExec: context.Args(), Quiet: context.Bool("q")}, configuration).Run(context.Args())
	}})

	customCommands := configuration.ListCommands()
	for key, value := range customCommands {
		commands = append(commands, cli.Command{Name: key, Action: customCommand(value, configuration), Flags: quietFlag})
	}
	return commands
}

func customCommand(execute string, configuration Repositories) func(*cli.Context) {
	return func(context *cli.Context) {
		NewRunner(&command.Run{ToExec: strings.Split(execute, " "), Quiet: context.Bool("q")}, configuration).Run(context.Args())
	}
}

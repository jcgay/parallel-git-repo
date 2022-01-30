package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/fatih/color"
	"github.com/jcgay/parallel-git-repo/version"
	"github.com/mitchellh/go-homedir"
	"github.com/pelletier/go-toml"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var home string

var (
	quiet        bool
	printVersion bool
)

const help = `NAME:
  Parallel Git Repositories - Execute commands on multiple Git repositories in parallel!

USAGE:
  ./parallel-git-repo [global options] command [arguments...]

VERSION:
  %s (%s)

COMMANDS:
%s
GLOBAL OPTIONS:
  -h	show help
`

var ok = color.New(color.FgGreen).SprintFunc()("✔")
var ko = color.New(color.FgRed).SprintFunc()("✘")

func main() {
	if home == "" {
		var err error
		home, err = homedir.Dir()
		if err != nil {
			log.Fatalf("Cannot read user home directory.\n%v", err)
		}
	}

	flag.BoolVar(&quiet, "q", false, "do not print stdout commands result, only stderr will be shown")
	flag.BoolVar(&printVersion, "v", false, "print current version")

	var group string
	flag.StringVar(&group, "g", "default", "execute command for a specific repositories group")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, help, version.VERSION, version.GITCOMMIT, listCommands())
		flag.PrintDefaults()
	}

	flag.Parse()

	if printVersion {
		fmt.Printf("version: %s (%s)", version.VERSION, version.GITCOMMIT)
	} else {
		configuration := newConfiguration(home)
		if os.Args[1] == "list" {
			repos := configuration.ListRepositories()
			for key, repos := range repos {
				fmt.Printf("%s:\n", key)
				for _, repo := range repos {
					fmt.Printf("  - %s\n", repo)
				}
			}
		} else {
			runCommand(configuration, withoutFlags(os.Args[1:]), group)
		}
	}
}

func listCommands() string {
	config, err := tryNewConfiguration(home)
	commands := make(map[string]string)
	if err == nil {
		commands = config.ListCommands()
	}

	maxSize := 3
	for key := range commands {
		if size := len(key); size > maxSize {
			maxSize = size
		}
	}

	result := fmt.Sprintf("  %-"+strconv.Itoa(maxSize)+"s	%s\n", "run", "run an arbitrary command")
	result += fmt.Sprintf("  %-"+strconv.Itoa(maxSize)+"s	%s\n", "list", "list repositories where command will be run")
	for key, value := range commands {
		result += fmt.Sprintf("  %-"+strconv.Itoa(maxSize)+"s	%s\n", key, value)
	}

	return result
}

func runCommand(config *configuration, args []string, group string) {
	commandName := args[0]
	var toExec []string
	if commandName == "run" {
		toExec = args[1:]
	} else {
		toExec = strings.Split(config.ListCommands()[commandName], " ")
	}

	runner := newRunner(&run{ToExec: toExec, Quiet: quiet}, config)
	runner.Run(args[1:], group)
}

func withoutFlags(args []string) []string {
	for i, value := range args {
		if !strings.HasPrefix(value, "-") {
			return args[i:]
		}
	}
	return args
}

type repositories interface {
	ListRepositories() map[string][]string
}

type configuration struct {
	content *toml.Tree
}

func newConfiguration(homeDir string) *configuration {
	config, err := tryNewConfiguration(homeDir)
	if err != nil {
		log.Fatalf("Can't read configuration file %s/.parallel-git-repositories, verify that the file is valid...\n%v", homeDir, err)
	}
	return config
}

func tryNewConfiguration(homeDir string) (*configuration, error) {
	config, err := toml.LoadFile(homeDir + "/.parallel-git-repositories")
	if err != nil {
		return nil, err
	}
	return &configuration{config}, nil
}

func (config *configuration) ListRepositories() map[string][]string {
	repos := config.content.Get("repositories").(*toml.Tree)
	result := make(map[string][]string)
	for _, key := range repos.Keys() {
		result[key] = toStringArray(repos.Get(key).([]interface{}))
	}
	return result
}

func toStringArray(values []interface{}) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.(string)
	}
	return result
}

func (config *configuration) ListCommands() map[string]string {
	all := config.content.Get("commands").(*toml.Tree)
	result := make(map[string]string)
	for _, key := range all.Keys() {
		result[key] = all.Get(key).(string)
	}
	return result
}

type runnableCommand interface {
	Executable() string
	Options() []string
	Output(output string, err error) string
}

type runner struct {
	runnableCommand

	repos  repositories
	writer io.Writer
}

func newRunner(command runnableCommand, repos repositories) *runner {
	return &runner{
		runnableCommand: command,
		repos:           repos,
		writer:          os.Stdout,
	}
}

func (runner *runner) Run(args []string, group string) {
	wg := sync.WaitGroup{}
	for key, repos := range runner.repos.ListRepositories() {
		if key == group {
			for _, repo := range repos {
				wg.Add(1)
				go func(repo string) {
					defer wg.Done()
					output := new(bytes.Buffer)

					command := exec.Command(runner.runnableCommand.Executable(), forwardArgs(runner.runnableCommand.Options(), args)...)
					command.Stdout = output
					command.Stderr = output
					command.Dir = repo
					err := command.Run()

					fmt.Fprintln(runner.writer, filepath.Base(repo)+": "+runner.runnableCommand.Output(strings.TrimSpace(output.String()), err))
				}(repo)
			}
		}
	}
	wg.Wait()
}

var option = regexp.MustCompile(`\$([0-9]+)`)

func forwardArgs(opts []string, args []string) []string {
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

type run struct {
	ToExec []string
	Quiet  bool
}

func (command *run) Executable() string {
	return command.ToExec[0]
}

func (command *run) Options() []string {
	return command.ToExec[1:]
}

func (command *run) Output(output string, err error) string {
	if err != nil {
		if output == "" {
			return fmt.Sprintf("%s\n  %v", ko, err)
		}
		return fmt.Sprintf("%s\n  %v\n  %s", ko, err, output)
	}
	if output == "" || command.Quiet {
		return ok
	}
	return fmt.Sprintf("%s\n  %s", ok, output)
}

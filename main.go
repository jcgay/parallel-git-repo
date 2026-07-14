package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/fatih/color"
	"github.com/mitchellh/go-homedir"
	"github.com/pelletier/go-toml"
)

var home string

// version is overridden at release time by GoReleaser (-X main.version=...).
// For `go install ...@version` builds it stays "dev" and buildInfo falls back
// to the module version stamped by the Go toolchain.
var version = "dev"

// buildInfo returns the version and short commit hash, read from the binary's
// build information so no ldflags gymnastics are needed for the commit.
func buildInfo() (string, string) {
	v := version
	var commit string
	if info, ok := debug.ReadBuildInfo(); ok {
		if v == "dev" && info.Main.Version != "" {
			v = info.Main.Version
		}
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				commit = setting.Value
				break
			}
		}
	}
	if len(commit) > 7 {
		commit = commit[:7]
	}
	return v, commit
}

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

	ver, commit := buildInfo()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, help, ver, commit, listCommands())
		flag.PrintDefaults()
	}

	flag.Parse()

	if printVersion {
		fmt.Printf("version: %s (%s)", ver, commit)
		return
	}

	// flag.Args() holds the positional arguments left after global flags have
	// been parsed. Reading os.Args[1] directly used to panic when the binary was
	// invoked without a command (e.g. `parallel-git-repo`).
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	configuration := newConfiguration(home)
	if args[0] == "list" {
		repos := configuration.ListRepositories()
		for key, repos := range repos {
			fmt.Printf("%s:\n", key)
			for _, repo := range repos {
				fmt.Printf("  - %s\n", repo)
			}
		}
	} else if runCommand(configuration, args, group) > 0 {
		os.Exit(1)
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

func runCommand(config *configuration, args []string, group string) int {
	commandName := args[0]
	var toExec []string
	if commandName == "run" {
		toExec = args[1:]
	} else {
		command, ok := config.ListCommands()[commandName]
		if !ok {
			log.Fatalf("Unknown command %q, run with -h to list available commands.", commandName)
		}
		toExec = strings.Split(command, " ")
	}

	runner := newRunner(&run{ToExec: toExec, Quiet: quiet}, config)
	return runner.Run(args[1:], group)
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
	result := make(map[string][]string)
	repos, ok := config.content.Get("repositories").(*toml.Tree)
	if !ok {
		return result
	}
	for _, key := range repos.Keys() {
		result[key] = toStringArray(repos.Get(key).([]interface{}))
	}
	return result
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func toStringArray(values []interface{}) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = value.(string)
	}
	return result
}

func (config *configuration) ListCommands() map[string]string {
	result := make(map[string]string)
	all, ok := config.content.Get("commands").(*toml.Tree)
	if !ok {
		return result
	}
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

func (runner *runner) Run(args []string, group string) int {
	var failures atomic.Int32
	wg := sync.WaitGroup{}
	all := runner.repos.ListRepositories()
	repos, found := all[group]
	if !found {
		fmt.Fprintf(os.Stderr, "Unknown group %q, available groups: %s\n", group, strings.Join(sortedKeys(all), ", "))
		return 1
	}
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
			if err != nil {
				failures.Add(1)
			}

			fmt.Fprintln(runner.writer, filepath.Base(repo)+": "+runner.runnableCommand.Output(strings.TrimSpace(output.String()), err))
		}(repo)
	}
	wg.Wait()

	return int(failures.Load())
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
				// Leave the placeholder untouched when no matching argument was
				// provided, rather than panicking on an out-of-range index.
				if index < 1 || index > len(args) {
					return substring
				}
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

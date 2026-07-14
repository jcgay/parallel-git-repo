package main

import (
	"bytes"
	"context"
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
	"time"

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
	jobs         int
	timeout      time.Duration
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
	flag.IntVar(&jobs, "j", 8, "maximum number of commands run in parallel")
	flag.DurationVar(&timeout, "timeout", 60*time.Second, "kill a command that runs longer than this (0 disables)")

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

	if args[0] == "add" {
		// Handled before newConfiguration so a missing or hand-broken config
		// file doesn't block the very command meant to write it.
		if err := addRepository(home, args[1:]); err != nil {
			log.Fatal(err)
		}
		return
	}

	configuration := newConfiguration(home)
	if args[0] == "list" {
		repos, err := filterGroup(configuration.ListRepositories(), group)
		if err != nil {
			log.Fatal(err)
		}
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

// addRepository registers a repository in the config file so onboarding no
// longer means hand-editing hidden TOML (a single typo there makes every later
// invocation fatal). The path defaults to the current directory and the group
// to "default"; a missing group is created and the rest of the file (other
// groups, commands) is preserved.
func addRepository(homeDir string, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	group := fs.String("g", "default", "group to add the repository to")
	if err := fs.Parse(args); err != nil {
		return err
	}

	path := "."
	if fs.NArg() > 0 {
		path = fs.Arg(0)
	}
	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	// .git is a directory in a normal clone but a file in worktrees and
	// submodules, so Stat (not IsDir) is the right check.
	if _, err := os.Stat(filepath.Join(path, ".git")); err != nil {
		return fmt.Errorf("%s is not a Git repository", path)
	}

	file := homeDir + "/.parallel-git-repositories"
	tree, err := toml.LoadFile(file)
	if os.IsNotExist(err) {
		tree, err = toml.Load("")
	}
	if err != nil {
		return err
	}

	members := (&configuration{tree}).ListRepositories()[*group]
	for _, repo := range members {
		if repo == path {
			return fmt.Errorf("%s is already in group %q", path, *group)
		}
	}
	tree.SetPath([]string{"repositories", *group}, append(members, path))

	out, err := tree.ToTomlString()
	if err != nil {
		return err
	}
	if err := os.WriteFile(file, []byte(out), 0644); err != nil {
		return err
	}
	fmt.Printf("Added %s to group %q\n", path, *group)
	return nil
}

// filterGroup narrows the group map to the requested group so `list` previews
// the same repositories `run` would touch, instead of always dumping every
// group. -g left at its default keeps the whole config; an explicit unknown
// group is an error, mirroring selectRepositories.
func filterGroup(all map[string][]string, group string) (map[string][]string, error) {
	if group == "default" {
		return all, nil
	}
	members, found := all[group]
	if !found {
		return nil, fmt.Errorf("Unknown group %q, available groups: %s", group, strings.Join(sortedKeys(all), ", "))
	}
	return map[string][]string{group: members}, nil
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
	result += fmt.Sprintf("  %-"+strconv.Itoa(maxSize)+"s	%s\n", "add", "register the current (or given) repository in a group")
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
		if needsShell(command) {
			// A naive split on spaces cannot express quoted arguments, pipes or
			// chaining, so route these through the shell. User arguments arrive as
			// the shell's "$@", matching the $@ placeholder plain commands use.
			if !strings.Contains(command, "$@") {
				command += ` "$@"`
			}
			// The trailing $@ is a placeholder token forwardArgs expands into the
			// shell's positional parameters ($1, $2, "$@") after the sh name.
			toExec = []string{"/bin/sh", "-c", command, "sh", "$@"}
		} else {
			toExec = strings.Split(command, " ")
		}
	}

	runner := newRunner(&run{ToExec: toExec, Quiet: quiet}, config)
	runner.jobs = jobs
	runner.timeout = timeout
	return runner.Run(args[1:], group)
}

// needsShell reports whether a configured command relies on shell features
// (quotes, pipes, chaining, redirection, subshells) that a plain space-split
// cannot honour. The $N/$@ placeholders are deliberately excluded so plain
// commands keep the faster direct-exec path.
func needsShell(command string) bool {
	return strings.ContainsAny(command, "|&;<>()`\"'")
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

	repos   repositories
	writer  io.Writer
	jobs    int
	timeout time.Duration
}

func newRunner(command runnableCommand, repos repositories) *runner {
	return &runner{
		runnableCommand: command,
		repos:           repos,
		writer:          os.Stdout,
		jobs:            8,
	}
}

func (runner *runner) Run(args []string, group string) int {
	var failures atomic.Int32
	wg := sync.WaitGroup{}
	repos, err := selectRepositories(runner.repos.ListRepositories(), group)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// forwardArgs is deterministic, so compute the argument list once instead of
	// once per goroutine.
	argv := forwardArgs(runner.runnableCommand.Options(), args)

	// Bound the number of concurrent child processes: without a limit, a large
	// group spawns one git process per repository at once, thrashing disk and
	// tripping server-side limits on concurrent SSH connections.
	limit := runner.jobs
	if limit < 1 {
		limit = 1
	}
	sem := make(chan struct{}, limit)

	for _, repo := range repos {
		wg.Add(1)
		go func(repo string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			output := new(bytes.Buffer)

			// Time the timeout from when the command actually starts, not when it
			// was queued, so repos waiting on the semaphore don't burn their budget.
			ctx := context.Background()
			if runner.timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, runner.timeout)
				defer cancel()
			}

			command := exec.CommandContext(ctx, runner.runnableCommand.Executable(), argv...)
			// Stop git blocking on a credential prompt (it reads /dev/tty even when
			// Stdin isn't wired); it fails fast instead. No effect on other commands.
			command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
			command.Stdout = output
			command.Stderr = output
			command.Dir = repo
			err := command.Run()
			if ctx.Err() == context.DeadlineExceeded {
				err = fmt.Errorf("timed out after %s", runner.timeout)
			}
			if err != nil {
				failures.Add(1)
			}

			fmt.Fprintln(runner.writer, filepath.Base(repo)+": "+runner.runnableCommand.Output(strings.TrimSpace(output.String()), err))
		}(repo)
	}
	wg.Wait()

	return int(failures.Load())
}

// selectRepositories resolves a group specifier into the deduplicated list of
// repositories to run against. The specifier may name several comma-separated
// groups, and the special value "all" selects every group. A repository listed
// in more than one selected group is kept once, in first-seen order. An unknown
// group name is an error.
func selectRepositories(all map[string][]string, group string) ([]string, error) {
	names := strings.Split(group, ",")
	if group == "all" {
		names = sortedKeys(all)
	}

	seen := make(map[string]struct{})
	var repos []string
	for _, name := range names {
		members, found := all[name]
		if !found {
			return nil, fmt.Errorf("Unknown group %q, available groups: %s", name, strings.Join(sortedKeys(all), ", "))
		}
		for _, repo := range members {
			if _, dup := seen[repo]; dup {
				continue
			}
			seen[repo] = struct{}{}
			repos = append(repos, repo)
		}
	}
	return repos, nil
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

package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func assertEqual(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestVersionFlagPrintsVersion(t *testing.T) {
	os.Args = []string{"parallel-git-repo", "-v"}

	original := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	main()
	w.Close()
	os.Stdout = original
	out, _ := io.ReadAll(r)

	// The exact version/commit depend on how the binary was built, so only
	// assert that the version line is produced.
	if !strings.Contains(string(out), "version: ") {
		t.Errorf("expected a version line, got %q", out)
	}
}

type SingleTempRepository struct {
	tempDir string
}

func (config *SingleTempRepository) ListRepositories() map[string][]string {
	config.tempDir, _ = os.MkdirTemp("", "parallel-git-repo")
	return map[string][]string{"default": {config.tempDir}}
}

func (config *SingleTempRepository) Dir() string {
	return filepath.Base(config.tempDir)
}

type PrintArgumentsCommand struct{}

func (command *PrintArgumentsCommand) Executable() string {
	return "echo"
}

func (command *PrintArgumentsCommand) Options() []string {
	return []string{"$@"}
}

func (command *PrintArgumentsCommand) Output(output string, err error) string {
	return output
}

func TestRunCommandWithArguments(t *testing.T) {
	output := new(bytes.Buffer)
	repos := &SingleTempRepository{}

	runner := newRunner(&PrintArgumentsCommand{}, repos)
	runner.writer = output
	runner.repos = repos
	runner.Run([]string{"first", "second"}, "default")

	assertEqual(t, output.String(), repos.Dir()+": first second\n")
}

type PrintArgumentsWithIndexCommand struct{}

func (command *PrintArgumentsWithIndexCommand) Executable() string {
	return "echo"
}

func (command *PrintArgumentsWithIndexCommand) Options() []string {
	return []string{"$1", "path/$10", "option=$3", "$4-$7"}
}

func (command *PrintArgumentsWithIndexCommand) Output(output string, err error) string {
	return output
}

func TestRunCommandWithIndexedArguments(t *testing.T) {
	output := new(bytes.Buffer)
	repos := &SingleTempRepository{}

	runner := newRunner(&PrintArgumentsWithIndexCommand{}, repos)
	runner.writer = output
	runner.repos = repos
	runner.Run([]string{"first", "second", "third", "4", "5", "6", "7", "8", "9", "10"}, "default")

	assertEqual(t, output.String(), repos.Dir()+": first path/10 option=third 4-7\n")
}

type FailingCommand struct{}

func (command *FailingCommand) Executable() string { return "false" }

func (command *FailingCommand) Options() []string { return []string{} }

func (command *FailingCommand) Output(output string, err error) string { return "" }

func TestRunReturnsFailureCount(t *testing.T) {
	repos := &SingleTempRepository{}
	runner := newRunner(&FailingCommand{}, repos)
	runner.writer = new(bytes.Buffer)

	if failures := runner.Run(nil, "default"); failures != 1 {
		t.Errorf("got %d failures, want 1", failures)
	}
}

func TestRunSucceedsWithZeroFailures(t *testing.T) {
	repos := &SingleTempRepository{}
	runner := newRunner(&PrintArgumentsCommand{}, repos)
	runner.writer = new(bytes.Buffer)

	if failures := runner.Run(nil, "default"); failures != 0 {
		t.Errorf("got %d failures, want 0", failures)
	}
}

func TestRunUnknownGroupIsAFailure(t *testing.T) {
	repos := &SingleTempRepository{}
	runner := newRunner(&PrintArgumentsCommand{}, repos)
	runner.writer = new(bytes.Buffer)

	if failures := runner.Run(nil, "does-not-exist"); failures == 0 {
		t.Error("got 0 failures for an unknown group, want non-zero")
	}
}

func TestForwardArgsLeavesPlaceholderWhenArgumentIsMissing(t *testing.T) {
	result := forwardArgs([]string{"$1", "$3"}, []string{"only-first"})

	assertEqual(t, result[0], "only-first")
	assertEqual(t, result[1], "$3")
}

func TestListRepositories(t *testing.T) {
	dir := t.TempDir()
	config := `
[repositories]
  default = [
    "/Users/jcgay/dev/maven-notifier",
    "/Users/jcgay/dev/maven-color"
  ]
  others = [
    "/Users/jcgay/dev/gradle-notifier",
    "/Users/jcgay/dev/buildplan-maven-plugin",
  ]
`
	os.WriteFile(dir+"/.parallel-git-repositories", []byte(config), 0644)
	repos := newConfiguration(dir)

	result := repos.ListRepositories()

	if len(result) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(result))
	}
	assertEqual(t, result["default"][0], "/Users/jcgay/dev/maven-notifier")
	assertEqual(t, result["default"][1], "/Users/jcgay/dev/maven-color")
	assertEqual(t, result["others"][0], "/Users/jcgay/dev/gradle-notifier")
	assertEqual(t, result["others"][1], "/Users/jcgay/dev/buildplan-maven-plugin")
}

func TestListCommands(t *testing.T) {
	dir := t.TempDir()
	config := `
[repositories]
  default = [
    "/Users/jcgay/dev/maven-notifier"
  ]
[commands]
  pull = "git pull"
  current-branch = "git symbolic-ref --short HEAD"
`
	os.WriteFile(dir+"/.parallel-git-repositories", []byte(config), 0644)
	commands := newConfiguration(dir)

	result := commands.ListCommands()

	if len(result) != 2 {
		t.Fatalf("expected 2 commands, got %d", len(result))
	}
	assertEqual(t, result["pull"], "git pull")
	assertEqual(t, result["current-branch"], "git symbolic-ref --short HEAD")
}

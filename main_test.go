package main

import (
	"bytes"
	"fmt"
	"github.com/assertgo/assert"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func Example() {
	os.Args = []string{"parallel-git-repo", "-v"}

	main()

	// Output:
	// version: unknown-snapshot (unknown-commit)
}

type SingleTempRepository struct {
	tempDir string
}

func (config *SingleTempRepository) ListRepositories() map[string][]string {
	config.tempDir, _ = ioutil.TempDir("", "parallel-git-repo")
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

	assert := assert.New(t)
	assert.ThatString(output.String()).IsEqualTo(repos.Dir() + ": first second\n")
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

	assert := assert.New(t)
	assert.ThatString(output.String()).IsEqualTo(repos.Dir() + ": first path/10 option=third 4-7\n")
}

func TestListRepositories(t *testing.T) {
	dir, _ := ioutil.TempDir("", "ParallelGitReposListRepositories")
	defer os.RemoveAll(dir)
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
	ioutil.WriteFile(dir+"/.parallel-git-repositories", []byte(config), 0644)
	repos := newConfiguration(dir)

	result := repos.ListRepositories()

	assert := assert.New(t)
	assert.ThatBool(len(result) == 2).IsTrue()
	assert.ThatString(result["default"][0]).IsEqualTo("/Users/jcgay/dev/maven-notifier")
	assert.ThatString(result["default"][1]).IsEqualTo("/Users/jcgay/dev/maven-color")
	assert.ThatString(result["others"][0]).IsEqualTo("/Users/jcgay/dev/gradle-notifier")
	assert.ThatString(result["others"][1]).IsEqualTo("/Users/jcgay/dev/buildplan-maven-plugin")
}

func TestListCommands(t *testing.T) {
	dir, _ := ioutil.TempDir("", "ParallelGitReposListCommands")
	defer os.RemoveAll(dir)
	config := `
[repositories]
  default = [
    "/Users/jcgay/dev/maven-notifier"
  ]
[commands]
  pull = "git pull"
  current-branch = "git symbolic-ref --short HEAD"
`
	ioutil.WriteFile(dir+"/.parallel-git-repositories", []byte(config), 0644)
	commands := newConfiguration(dir)

	result := commands.ListCommands()

	assert := assert.New(t)
	assert.ThatBool(len(result) == 2).IsTrue()
	assert.ThatString(result["pull"]).IsEqualTo("git pull")
	assert.ThatString(result["current-branch"]).IsEqualTo("git symbolic-ref --short HEAD")
}

type PrintArguments struct{}

func (command *PrintArguments) Executable() string {
	return "echo"
}

func (command *PrintArguments) Options() []string {
	return []string{"$@"}
}

func (command *PrintArguments) Output(output string, err error) string {
	return output
}

func TestRunCommandWithAllGroupsActivated(t *testing.T) {
	dir, _ := ioutil.TempDir("", "RunCommandWithAllGroupsActivated")
	defer os.RemoveAll(dir)
	config := `
[repositories]
        default = [
                "/Users/jcgay/dev/maven-notifier"
        ]
        others = [
                "/Users/jcgay/dev/gradle-notifier"
        ]
`

	ioutil.WriteFile(dir+"/.parallel-git-repositories", []byte(config), 0644)
	repos := newConfiguration(dir)

	allGroups = true
	defer reset()

	output := new(bytes.Buffer)
	runner := newRunner(&PrintArguments{}, repos)
	runner.writer = output
	runner.repos = repos
	runner.Run([]string{"something"}, "default")

	assert := assert.New(t)
	result := strings.Split(output.String(), "\n")

	fmt.Println(output.String())
	assert.ThatBool(contains(result, "maven-notifier: something")).IsTrue()
	assert.ThatBool(contains(result, "gradle-notifier: something")).IsTrue()
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func reset() {
	allGroups = false
}

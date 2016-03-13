package main

import (
	"bytes"
	"github.com/assertgo/assert"
	"github.com/codegangsta/cli"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func ExampleShowVersion() {
	os.Args = []string{"parallel-git-repo", "-v"}

	main()

	// Output:
	// Parallel Git Repositories version 0.0.0
}

type SingleTempRepository struct {
	tempDir string
}

func (this *SingleTempRepository) List() []string {
	this.tempDir, _ = ioutil.TempDir("", "parallel-git-repo")
	return []string{this.tempDir}
}

func (this *SingleTempRepository) Dir() string {
	return filepath.Base(this.tempDir)
}

type PrintArgumentsCommand struct{}

func (command *PrintArgumentsCommand) Executable() string {
	return "echo"
}

func (command *PrintArgumentsCommand) Options() []string {
	return []string{"$@"}
}

func (command *PrintArgumentsCommand) Output(output string) string {
	return output
}

func TestRunCommandWithArguments(t *testing.T) {
	output := new(bytes.Buffer)
	repos := &SingleTempRepository{}

	runner := NewRunner(&PrintArgumentsCommand{})
	runner.writer = output
	runner.repos = repos
	runner.Run(cli.Args{"first", "second"})

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

func (command *PrintArgumentsWithIndexCommand) Output(output string) string {
	return output
}

func TestRunCommandWithIndexedArguments(t *testing.T) {
	output := new(bytes.Buffer)
	repos := &SingleTempRepository{}

	runner := NewRunner(&PrintArgumentsWithIndexCommand{})
	runner.writer = output
	runner.repos = repos
	runner.Run(cli.Args{"first", "second", "third", "4", "5", "6", "7", "8", "9", "10"})

	assert := assert.New(t)
	assert.ThatString(output.String()).IsEqualTo(repos.Dir() + ": first path/10 option=third 4-7\n")
}

func TestListRepositories(t *testing.T) {
	dir, _ := ioutil.TempDir("", "ParallelGitReposListRepositories")
	defer os.RemoveAll(dir)
	config := `repositories:
  - /Users/jcgay/dev/maven-notifier
  - /Users/jcgay/dev/maven-color
`
	ioutil.WriteFile(dir+"/.parallel-git-repositories", []byte(config), 0644)
	repos := RegisteredRepositories{dir}

	result := repos.List()

	assert := assert.New(t)
	assert.ThatBool(len(result) == 2).IsTrue()
	assert.ThatString(result[0]).IsEqualTo("/Users/jcgay/dev/maven-notifier")
	assert.ThatString(result[1]).IsEqualTo("/Users/jcgay/dev/maven-color")
}

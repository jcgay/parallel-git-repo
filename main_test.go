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

func ExampleRunCommandForMultipleRepositories() {
	os.Args = []string{"parallel-git-repo", "echo"}

	main()

	// Output:
	// maven-notifier: /Users/jcgay/dev/maven-notifier
	// maven-color: /Users/jcgay/dev/maven-color
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

func TestRunCommandWithArguments(t *testing.T) {
	assert := assert.New(t)
	output := new(bytes.Buffer)
	repos := &SingleTempRepository{}

	runner := NewRunner(&PrintArgumentsCommand{})
	runner.writer = output
	runner.repos = repos
	runner.Run(cli.Args{"first", "second"})

	assert.ThatString(output.String()).IsEqualTo(repos.Dir() + ": first second\n")
}

type PrintArgumentsWithIndexCommand struct{}

func (command *PrintArgumentsWithIndexCommand) Executable() string {
	return "echo"
}

func (command *PrintArgumentsWithIndexCommand) Options() []string {
	return []string{"$1", "path/$10", "option=$3", "$4-$7"}
}

func TestRunCommandWithIndexedArguments(t *testing.T) {
	assert := assert.New(t)
	output := new(bytes.Buffer)
	repos := &SingleTempRepository{}

	runner := NewRunner(&PrintArgumentsWithIndexCommand{})
	runner.writer = output
	runner.repos = repos
	runner.Run(cli.Args{"first", "second", "third", "4", "5", "6", "7", "8", "9", "10"})

	assert.ThatString(output.String()).IsEqualTo(repos.Dir() + ": first path/10 option=third 4-7\n")
}

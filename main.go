package main

import (
	"io"
	"os"
	"os/exec"
)

func main() {
	run(os.Args, os.Stdout, os.Stderr, &RegisteredRepositories{})
}

type Repositories interface {
	List() []string
}

type RegisteredRepositories struct{}

func (repos *RegisteredRepositories) List() []string {
	return []string{"/Users/jcgay/dev/maven-notifier", "/Users/jcgay/dev/maven-color"}
}

func run(args []string, output io.Writer, error io.Writer, repos Repositories) {
	for _, repo := range repos.List() {
		command := exec.Command("echo", repo)
		command.Stdout = output
		command.Stderr = error
		command.Dir = repo
		command.Run()
	}
}

package main

import (
	"github.com/codegangsta/cli"
	"io"
	"os"
	"os/exec"
)

func main() {
	app := cli.NewApp()
	app.Name = "Parallel Git Repositories"
	app.Usage = "Execute commands on multiple Git repositories at the same time!"
	app.Commands = buildCommands()
	app.Run(os.Args)
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

func buildCommands() []cli.Command {
	commands := make([]cli.Command, 1)
	commands = append(commands, cli.Command{
		Name: "echo",
		Action: func(context *cli.Context) {
			run(os.Args, os.Stdout, os.Stderr, &RegisteredRepositories{})
		},
	})
	return commands
}

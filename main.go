package main

import (
	"io"
	"os"
	"os/exec"
)

func main() {
	run(os.Args, os.Stdout, os.Stderr)
}

func run(args []string, output io.Writer, error io.Writer) {
	command := exec.Command("echo", "1")
	command.Stdout = output
	command.Stderr = error
	command.Dir = "/Users/jcgay/dev/maven-notifier"
	command.Run()
}

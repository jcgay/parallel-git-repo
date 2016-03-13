package command

import (
	"fmt"
)

type GitStatus struct{}

func (command *GitStatus) Executable() string {
	return "git"
}

func (command *GitStatus) Options() []string {
	return []string{"status", "-s"}
}

func (command *GitStatus) Output(output string, errOutput string, err error) string {
	if err != nil {
		return fmt.Sprintf("%v\n  %s", err, errOutput)
	}

	if output == "" {
		return ok
	}

	return fmt.Sprintf("%s\n%s", ko, output)
}

package command

import "fmt"

type GitCheckout struct{}

func (command *GitCheckout) Executable() string {
	return "git"
}

func (command *GitCheckout) Options() []string {
	return []string{"checkout", "$@"}
}

func (command *GitCheckout) Output(output string, errOutput string, err error) string {
	if err != nil {
		return fmt.Sprintf("%s\n  %v\n%s", ko, err, errOutput)
	}

	return ok
}

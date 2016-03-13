package command

import "fmt"

type GitShowCurrentBranch struct{}

func (command *GitShowCurrentBranch) Executable() string {
	return "git"
}

func (command *GitShowCurrentBranch) Options() []string {
	return []string{"symbolic-ref", "--short", "HEAD"}
}

func (command *GitShowCurrentBranch) Output(output string, errOutput string, err error) string {
	if err != nil {
		return fmt.Sprintf("%v\n  %s", err, errOutput)
	}
	return output
}

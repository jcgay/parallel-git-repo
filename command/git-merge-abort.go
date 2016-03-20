package command

import "fmt"

type GitMergeAbort struct{}

func (command *GitMergeAbort) Executable() string {
	return "git"
}

func (command *GitMergeAbort) Options() []string {
	return []string{"merge", "--abort"}
}

func (command *GitMergeAbort) Output(output string, errOutput string, err error) string {
	if err == nil {
		return ok
	}
	return fmt.Sprintf("%s\n %s", ko, errOutput)
}

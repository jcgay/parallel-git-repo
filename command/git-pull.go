package command

type GitPull struct{}

func (command *GitPull) Executable() string {
	return "git"
}

func (command *GitPull) Options() []string {
	return []string{"pull", "--rebase=preserve"}
}

func (command *GitPull) Output(output string) string {
	return output
}

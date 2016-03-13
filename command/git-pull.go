package command

type GitPull struct{}

func (command *GitPull) Executable() string {
	return "git"
}

func (command *GitPull) Options() []string {
	return []string{"pull", "--quiet", "--rebase=preserve"}
}

func (command *GitPull) Output(output string, errOutput string, err error) string {
	return tickOutput(err)
}

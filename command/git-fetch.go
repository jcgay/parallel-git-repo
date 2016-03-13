package command

type GitFetch struct{}

func (command *GitFetch) Executable() string {
	return "git"
}

func (command *GitFetch) Options() []string {
	return []string{"fetch", "-p"}
}

func (command *GitFetch) Output(output string, errOutput string, err error) string {
	return tickOutput(err)
}

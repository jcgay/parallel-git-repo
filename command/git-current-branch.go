package command

type GitShowCurrentBranch struct{}

func (command *GitShowCurrentBranch) Executable() string {
	return "git"
}

func (command *GitShowCurrentBranch) Options() []string {
	return []string{"symbolic-ref", "--short", "HEAD"}
}

func (command *GitShowCurrentBranch) Output(output string) string {
	return output
}

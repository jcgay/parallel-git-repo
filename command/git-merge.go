package command

type GitMerge struct{}

func (command *GitMerge) Executable() string {
	return "git"
}

func (command *GitMerge) Options() []string {
	return []string{"merge", "$1"}
}

func (command *GitMerge) Output(output string) string {
	return output
}

package command

import "fmt"

type GitLlg struct{}

func (command *GitLlg) Executable() string {
	return "git"
}

func (command *GitLlg) Options() []string {
	return []string{"log", "--graph", "--pretty=tformat:%Cred%h%Creset -%C(auto)%d%Creset %s %Cgreen(%an %cr)%Creset", "@{u}.."}
}

func (command *GitLlg) Output(output string, errOutput string, err error) string {
	if err != nil {
		return fmt.Sprintf("%s\n %s", ko, errOutput)
	}

	if output == "" {
		return ok
	}

	return fmt.Sprintf("%s\n%s", ok, output)
}

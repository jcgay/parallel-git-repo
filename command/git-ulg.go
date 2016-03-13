package command

import "fmt"

type GitUlg struct{}

func (command *GitUlg) Executable() string {
	return "git"
}

func (command *GitUlg) Options() []string {
	return []string{"log", "--graph", "--pretty=tformat:%Cred%h%Creset -%C(auto)%d%Creset %s %Cgreen(%an %cr)%Creset", "..@{u}"}
}

func (command *GitUlg) Output(output string, errOutput string, err error) string {
	if err != nil {
		return fmt.Sprintf("%s\n %s", ko, errOutput)
	}

	if output == "" {
		return ok
	}

	return fmt.Sprintf("%s\n%s", ok, output)
}

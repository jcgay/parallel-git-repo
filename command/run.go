package command

import "fmt"

type Run struct {
	ToExec []string
	Quiet  bool
}

func (command *Run) Executable() string {
	return command.ToExec[0]
}

func (command *Run) Options() []string {
	return command.ToExec[1:]
}

func (command *Run) Output(output string, err error) string {
	if err != nil {
		if output == "" {
			return fmt.Sprintf("%s\n  %v", ko, err)
		}
		return fmt.Sprintf("%s\n  %v\n  %s", ko, err, output)
	}
	if output == "" || command.Quiet {
		return ok
	}
	return fmt.Sprintf("%s\n  %s", ok, output)
}

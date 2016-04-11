package command

import "fmt"

type Run struct {
	ToExec []string
}

func (command *Run) Executable() string {
	return command.ToExec[0]
}

func (command *Run) Options() []string {
	return command.ToExec[1:]
}

func (command *Run) Output(output string, errOutput string, err error) string {
	if err != nil {
		return fmt.Sprintf("%v\n  %s", err, errOutput)
	}
	if output == "" {
		return ok
	}
	return output
}

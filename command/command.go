package command

import "github.com/fatih/color"

var ok = color.New(color.FgGreen).SprintFunc()("✔")
var ko = color.New(color.FgRed).SprintFunc()("✘")

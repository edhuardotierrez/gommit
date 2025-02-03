package colors

import "github.com/fatih/color"

var (
	ErrorOutput   = color.New(color.FgRed).PrintfFunc()
	InfoOutput    = color.New(color.FgCyan).PrintfFunc()
	DescOutput    = color.New(color.FgHiCyan).PrintfFunc()
	TextOutput    = color.New(color.FgWhite).PrintfFunc()
	SuccessOutput = color.New(color.FgGreen).PrintfFunc()
	WarningOutput = color.New(color.FgYellow).PrintfFunc()
)

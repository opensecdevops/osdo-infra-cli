package ui

import "github.com/fatih/color"

// Color helpers - centralized to avoid redeclaration
var (
	grayColor   = color.New(color.FgHiBlack)
	grayString  = grayColor.SprintFunc()
	grayStringf = grayColor.SprintfFunc()
)

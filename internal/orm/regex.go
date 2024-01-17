package internal

import (
	"regexp"
	"strings"
)

type regex struct {
	CREATE_TABLE_ROWS func(string) string
	FIELD_NAME        func(string) string
}

// Internal regular expressions used in code as functions to ensure
// code readability and ease to use.
var Regexp = regex{
	CREATE_TABLE_ROWS: func(value string) string {
		return regexp.MustCompile(
			"CREATE TABLE `\\w+` \\((.+)\\).+$",
		).FindStringSubmatch(strings.ReplaceAll(value, "\n", ""))[1]
	},
	FIELD_NAME: func(value string) string {
		return regexp.MustCompile(
			"`(\\w+)`",
		).FindStringSubmatch(value)[1]
	},
}

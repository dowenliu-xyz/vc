package sub

import (
	"regexp"
	"strings"
)

var legacySsShareUrlPattern = regexp.MustCompile("(?P<method>.+):(?P<pass>.+)@(?P<host>.+):(?P<port>\\d+)")

func divideStr(s string, sep string) (string, string) {
	parts := strings.SplitN(s, sep, 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return s, ""
}

package disposable

import (
	_ "embed"
	"strings"
)

//go:embed list.txt
var rawList string

var disposableSet map[string]struct{}

func init() {
	disposableSet = make(map[string]struct{})
	for _, line := range strings.Split(rawList, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			disposableSet[strings.ToLower(line)] = struct{}{}
		}
	}
}

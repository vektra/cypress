package cypress

import "strings"

func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		path = HomeDir + "/" + path[2:]
	}

	return path
}

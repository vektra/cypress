package elasticsearch

import "github.com/vektra/cypress/cli/commands"

func init() {
	commands.Add("elasticsearch:send", "write messages to elasticsearch", "", &Send{})
}

package plugins

import (
	_ "github.com/vektra/cypress/plugins/grep"
	_ "github.com/vektra/cypress/plugins/metrics"
	_ "github.com/vektra/cypress/plugins/postgresql"
	_ "github.com/vektra/cypress/plugins/s3"
	_ "github.com/vektra/cypress/plugins/spool"
	_ "github.com/vektra/cypress/plugins/statsd"
	_ "github.com/vektra/cypress/plugins/syslog"
)

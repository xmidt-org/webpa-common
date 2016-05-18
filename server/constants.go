package server

import (
	"time"
)

const (
	// DefaultServerName is the default value for the server name.  It's almost never
	// a good idea to use this default.  It's simply here to allow valid 0-values in structs.
	DefaultServerName string = "webpa"

	// DefaultPort is the default value for the port of the primary server
	DefaultPort uint16 = 8080

	// DefaultHealthCheckPort is the default value for the port on which health check listens
	DefaultHealthCheckPort uint16 = 8888

	// DefaultPprofPort is the default value for the port on which pprof listens
	DefaultPprofPort uint16 = 9999

	// DefaultHealthCheckInterval is the default interval on which health statistics
	// will be sent out
	DefaultHealthCheckInterval time.Duration = time.Duration(time.Second * 60)

	// healthSuffix is the string appended to server name's to produce the health server name
	healthSuffix string = ".health"

	// pprofSuffix is the string appended to server name's to produce the pprof server name
	pprofSuffix string = ".pprof"
)

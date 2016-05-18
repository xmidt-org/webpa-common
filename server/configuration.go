package server

import (
	"encoding/json"
	"github.com/Comcast/webpa-common/types"
	"io/ioutil"
	"os"
)

// Configuration provides the basic configuration options common to all WebPA servers.
type Configuration struct {
	// ServerName is the human-readable name for this server.  This will be used as the name of
	// the internal logger.  Note that this is exposed via JSON, but doesn't have to be supplied
	// from a configuration file.  Applications can hardcode it at will.
	ServerName string `json:"serverName"`

	// Port is the primary port for this server
	Port uint16 `json:"port"`

	// HealthCheckPort is the port used for the health check service.  This service
	// is always HTTP.
	HealthCheckPort uint16 `json:"hcport"`

	// HealthCheckInterval is the interval at which health logging is dispatched
	HealthCheckInterval types.Duration `json:"hcInterval"`

	// PprofPort is the port used for pprof.  This service
	// is always HTTP.
	PprofPort uint16 `json:"pprofport"`

	// CertificateFile is the path to the file containing the certificate for HTTPS.
	// This only applies to the primary server listening on Port.
	CertificateFile string `json:"cert"`

	// KeyFile is the path to the file containing the key for HTTPS.
	// This only applies to the primary server listening on Port.
	KeyFile string `json:"key"`
}

// ReadConfigurationFile provides the standard logic for reading a JSON
// configuration file and returning the appropriate object.  This method does
// not assume any configuration type, but most often it will be Configuration
// or a type that embeds Configuration.
func ReadConfigurationFile(filename string, configuration interface{}) (err error) {
	if _, err = os.Lstat(filename); err == nil {
		buffer, err := ioutil.ReadFile(filename)
		if err == nil {
			err = json.Unmarshal(buffer, configuration)
		}
	}

	return
}

package server

import (
	"encoding/json"
	"fmt"
	"github.com/Comcast/webpa-common/types"
	"io/ioutil"
	"os"
	"time"
)

// Configuration provides the basic configuration options common to all WebPA servers.
type Configuration struct {
	// Port is the primary port for this server
	Port uint16 `json:"port"`

	// HealthCheckPort is the port used for the health check service.  This service
	// is always HTTP.
	HealthCheckPort uint16 `json:"hcport"`

	// HCInterval is the interval at which health logging is dispatched
	HCInterval types.Duration `json:"hcInterval"`

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

// PrimaryAddress returns the listen address for the primary server, i.e.
// the server that listens on c.Port.
func (c *Configuration) PrimaryAddress() string {
	port := DefaultPort
	if c.Port > 0 {
		port = c.Port
	}

	return fmt.Sprintf(":%d", port)
}

// HealthAddress returns the listen address for the health server
func (c *Configuration) HealthAddress() string {
	port := DefaultHealthCheckPort
	if c.HealthCheckPort > 0 {
		port = c.HealthCheckPort
	}

	return fmt.Sprintf(":%d", port)
}

// HealthCheckInterval returns the period between health updates
func (c *Configuration) HealthCheckInterval() time.Duration {
	if c.HCInterval > 0 {
		return time.Duration(c.HCInterval)
	} else {
		return DefaultHealthCheckInterval
	}
}

// PprofAddress returns the listen address for the pprof server
func (c *Configuration) PprofAddress() string {
	port := DefaultPprofPort
	if c.PprofPort > 0 {
		port = c.PprofPort
	}

	return fmt.Sprintf(":%d", port)
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

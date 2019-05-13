package client

import (
	"reflect"
	"time"
)

type DialerConfig struct {
	Timeout time.Duration
	// Deadline      time.Time
	// LocalAddr     net.Addr
	FallbackDelay time.Duration
	KeepAlive     time.Duration
	// Resolver *net.Resolver
}

func (d *DialerConfig) timeOut() time.Duration {
	if d != nil && d.Timeout > 0 {
		return d.Timeout
	}

	return 0
}

/* TODO: include deadline
func (d *DialerConfig) deadline() time.Time {
	if d != nil && d.Deadline > 0 {
		return d.Deadline
	}

	return time.Time(0)
}
*/

/*
func (d *DialerConfig) localAddr() net.Addr {
	if d != nil && d.LocalAddr != nil {
		return d.LocalAddr
	}

	return nil
}
*/

func (d *DialerConfig) fallbackDelay() time.Duration {
	if d != nil && d.FallbackDelay > 0 {
		return d.FallbackDelay
	}

	return 0
}

// TODO: this may need to negative not ZERO.
func (d *DialerConfig) keepAlive() time.Duration {
	if d != nil && d.KeepAlive > 0 {
		return d.KeepAlive
	}

	return 0
}

func (d *DialerConfig) IsEmpty() bool {
	return reflect.DeepEqual(d, (DialerConfig{}))
}

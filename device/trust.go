package device

import "strconv"

// Trust indicates the level of trust for a device, based primarily around the
// credentials establish at connection time.
type Trust int

const (
	Untrusted Trust = 0
	Trusted   Trust = 1
)

func (t Trust) String() string {
	return strconv.Itoa(int(t))
}

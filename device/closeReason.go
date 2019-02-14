package device

// CloseReason exposes metadata around why a particular device was closed
type CloseReason struct {
	// Err is the optional field that specifies the underlying error that occurred, such as
	// an I/O error.  If nil, the close reason is assumed to be due to application logic, e.g. a rehash
	Err error

	// Text is the required field indicating a JSON-friendly value describing the reason for closure.
	Text string
}

func (c CloseReason) String() string {
	errText := "*no error*"
	if c.Err != nil {
		errText = c.Err.Error()
	}

	return errText + ":" + c.Text
}

package logging

import (
	"bytes"
	"io"
	"github.com/go-kit/kit/log"
	"time"
	"fmt"
	"strings"
)

type reformatLogger struct {
	baseTimestamp time.Time
	w             io.Writer
}

// NewLogfmtLogger returns a logger that encodes keyvals to the Writer in
// logfmt format. Each log event produces no more than one call to w.Write.
// The passed Writer must be safe for concurrent use by multiple goroutines if
// the returned Logger will be used concurrently.
func NewReformatLogger(w io.Writer) log.Logger {
	return log.NewSyncLogger(&reformatLogger{time.Now(), w})
}

func (l reformatLogger) Log(keyvals ...interface{}) error {
	var buf bytes.Buffer
	data := mapKeyVals(keyvals)

	// Write the header to appear ERROR[00000]	oh now it broke
	if data.level == "" {
		data.level = "info"
	}
	if data.Time.IsZero() {
		data.Time = time.Now()
	}

	buf.WriteString(fmt.Sprintf("%s[%05d]\t %s\t\t", string([]rune(strings.ToUpper(data.level))[0:4]), int(data.Time.Sub(l.baseTimestamp).Seconds()), data.msg))

	for key, value := range data.fieldMap {
		buf.WriteString(fmt.Sprintf("%s=%s ", key, value))
	}

	// Add newline to the end of the buffer
	buf.WriteString("\n")

	// The Logger interface requires implementations to be safe for concurrent
	// use by multiple goroutines. For this implementation that means making
	// only one call to l.w.Write() for each call to Log.
	if _, err := l.w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

type reformatData struct {
	Time     time.Time
	msg      string
	level    string
	fieldMap map[string]interface{}
}

func mapKeyVals(keyvals []interface{}) reformatData {
	data := reformatData{}
	data.fieldMap = make(map[string]interface{})
	for i := 0; i < len(keyvals)-1; i += 2 {
		if getString(keyvals[i]) == "level" {
			data.level = getString(keyvals[i+1])
			continue
		}
		if getString(keyvals[i]) == "ts" {
			if t, ok := keyvals[i+1].(time.Time); ok {
				data.Time = t
			} else if t, err := time.Parse(time.RFC3339, getString(keyvals[i+1])); err == nil {
				data.Time = t
			} else {
				// We failed to parse it this should be close enough
				data.Time = time.Now()
			}
			continue
		}
		if getString(keyvals[i]) == "msg" {
			data.msg = getString(keyvals[i+1])
			continue
		}
		data.fieldMap[getString(keyvals[i])] = keyvals[i+1]

	}
	return data
}

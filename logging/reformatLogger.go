package logging

import (
	"bytes"
	"io"
	"github.com/go-kit/kit/log"
	"time"
	"fmt"
	"strings"
	"github.com/go-kit/kit/log/term"
	"errors"
)

const (
	Default = term.Color(iota)

	Black
	DarkRed
	DarkGreen
	Brown
	DarkBlue
	DarkMagenta
	DarkCyan
	Gray

	DarkGray
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White

	numColors
)

var (
	resetColorBytes = []byte("\x1b[39;49;22m")
	fgColorBytes    [][]byte
	bgColorBytes    [][]byte
)

func init() {
	// Default
	fgColorBytes = append(fgColorBytes, []byte("\x1b[39m"))
	bgColorBytes = append(bgColorBytes, []byte("\x1b[49m"))

	// dark colors
	for color := Black; color < DarkGray; color++ {
		fgColorBytes = append(fgColorBytes, []byte(fmt.Sprintf("\x1b[%dm", 30+color-Black)))
		bgColorBytes = append(bgColorBytes, []byte(fmt.Sprintf("\x1b[%dm", 40+color-Black)))
	}

	// bright colors
	for color := DarkGray; color < numColors; color++ {
		fgColorBytes = append(fgColorBytes, []byte(fmt.Sprintf("\x1b[%d;1m", 30+color-DarkGray)))
		bgColorBytes = append(bgColorBytes, []byte(fmt.Sprintf("\x1b[%d;1m", 40+color-DarkGray)))
	}
}

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

func colorVals(data reformatData) term.FgBgColor {

	switch data.level {
	case "debug":
		return term.FgBgColor{Fg: Gray}
	case "info":
		return term.FgBgColor{Fg: Blue}
	case "warn":
		return term.FgBgColor{Fg: Yellow}
	case "error":
		return term.FgBgColor{Fg: Red}
	case "crit":
		return term.FgBgColor{Fg: Gray, Bg: DarkRed}
	default:
		return term.FgBgColor{}
	}
}

func getColorBytes(isTerm bool, color term.FgBgColor) []byte {
	var buf bytes.Buffer

	if isTerm {
		if color.Fg != Default {
			buf.Write(fgColorBytes[color.Fg])
		}
		if color.Bg != Default {
			buf.Write(bgColorBytes[color.Bg])
		}
	}
	return buf.Bytes()
}

func (l reformatLogger) Log(keyvals ...interface{}) error {
	var buf bytes.Buffer
	data := mapKeyVals(keyvals)
	color := colorVals(data)

	isTerm := term.IsTerminal(l.w)

	// Write the header to appear ERRO[00000]	4 characters for column awesomeness
	if data.level == "" {
		data.level = "info"
	}
	if data.Time.IsZero() {
		data.Time = time.Now()
	}

	// Write Level
	buf.Write(getColorBytes(isTerm, color))
	buf.WriteString(string([]rune(strings.ToUpper(data.level))[0:4]))
	if isTerm {
		//reset
		buf.Write(resetColorBytes)
	}

	// Write Time
	buf.WriteString(fmt.Sprintf("[%05d] ", int(data.Time.Sub(l.baseTimestamp).Seconds())))

	// Write message
	buf.Write(getColorBytes(isTerm, color))
	if data.msg != "" {
		buf.WriteString(fmt.Sprintf("\t%s\t\t", data.msg))
	}
	if isTerm {
		buf.Write(resetColorBytes)
	}

	// Write Error
	if data.error != nil {
		buf.Write(getColorBytes(isTerm, term.FgBgColor{Bg: Red, Fg: Black}))
		buf.WriteString("ERR")
		if isTerm {
			buf.Write(resetColorBytes)
		}
		buf.WriteString(fmt.Sprintf("=%s\t", data.error.Error()))
	}

	// Write KeyValue's
	for key, value := range data.fieldMap {
		//key
		buf.Write(getColorBytes(isTerm, color))
		buf.WriteString(key)
		if isTerm {
			buf.Write(resetColorBytes)
		}

		//value
		buf.WriteString(fmt.Sprintf("=%v ", value))
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
	error    error
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
		if getString(keyvals[i]) == "err" || getString(keyvals[i]) == "error" {
			if err, ok := keyvals[i+1].(error); ok {
				data.error = err
				continue
			} else if errString := getString(keyvals[i+1]); len(strings.TrimSpace(errString)) > 0 {
				data.error = errors.New(errString)
				continue
			}

		}
		data.fieldMap[getString(keyvals[i])] = keyvals[i+1]

	}
	return data
}

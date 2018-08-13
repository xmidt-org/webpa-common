package logging

import (
	"bytes"
	"io"
	"github.com/go-kit/kit/log"
	"time"
	"fmt"
	"strings"
	"errors"
	"github.com/ttacon/chalk"
	"os"
	"golang.org/x/crypto/ssh/terminal"
)

const (
	nocolor = 0
	red     = 31
	green   = 32
	yellow  = 33
	blue    = 36
	gray    = 37
)

type TextFormatter struct {
	// Force disabling colors.
	DisableColors bool `json:"disable_colors"`

	// Disables the truncation of the level text to 4 characters.
	DisableLevelTruncation bool `json:"disable_level_truncation"`

	// Whether the logger's out is to a terminal
	isTerminal bool
}

type reformatLogger struct {
	baseTimestamp time.Time
	w             io.Writer
	formatter     *TextFormatter
}

// NewLogfmtLogger returns a logger that encodes keyvals to the Writer in
// logfmt format. Each log event produces no more than one call to w.Write.
// The passed Writer must be safe for concurrent use by multiple goroutines if
// the returned Logger will be used concurrently.
func NewReformatLogger(w io.Writer, formatter *TextFormatter) log.Logger {
	formatter.init(w)

	return log.NewSyncLogger(&reformatLogger{time.Now(), w, formatter})
}

func colorVals(data reformatData) chalk.Color {

	switch data.level {
	case "debug":
		return chalk.Green
	case "info":
		return chalk.Blue
	case "warn":
		return chalk.Yellow
	case "error":
		return chalk.Red
	default:
		return chalk.ResetColor
	}
}

func (t *TextFormatter) init(writer io.Writer) {
	t.isTerminal = checkIfTerminal(writer)
}

func checkIfTerminal(w io.Writer) bool {
	switch v := w.(type) {
	case *os.File:
		return terminal.IsTerminal(int(v.Fd()))
	default:
		return false
	}
}

func (t *TextFormatter) isColored() bool {
	return t.isTerminal && !t.DisableColors
}

func writeColor(buf *bytes.Buffer, color chalk.Color, formatter *TextFormatter, coloredText string) {
	if formatter.isColored() {
		fmt.Fprint(buf, color.Color(coloredText))
	} else {
		fmt.Fprint(buf, coloredText)
	}
}

func writeStyle(buf *bytes.Buffer, style chalk.Style, formatter *TextFormatter, styledText string) {
	if formatter.isColored() {
		fmt.Fprint(buf, style.Style(styledText))
	} else {
		fmt.Fprint(buf, styledText)
	}
}

func (t *TextFormatter) getColor(color chalk.Color) string {
	if !t.isColored() {
		return ""
	}
	return color.String()
}

func (l *reformatLogger) Log(keyvals ...interface{}) error {
	buf := &bytes.Buffer{}
	data := mapKeyVals(keyvals)
	color := colorVals(data)

	levelText := strings.ToUpper(data.level)
	if !l.formatter.DisableLevelTruncation {
		levelText = levelText[0:4]
	}

	// Write Level aka INFO, WARN
	writeColor(buf, color, l.formatter, levelText)
	fmt.Fprintf(buf, "[%05d] %-44s", int(data.time.Sub(l.baseTimestamp)/time.Second), data.msg)

	// if err, wright it
	if nil != data.error {
		writeStyle(buf, chalk.Black.NewStyle().WithBackground(chalk.Red), l.formatter, "ERR")
		fmt.Fprintf(buf, ":%#v ", data.error.Error())
	}

	// Write KeyValue's
	for key, value := range data.fieldMap {
		writeColor(buf, color, l.formatter, key)
		fmt.Fprintf(buf, "=%#v ", value)
	}

	// Add newline to the end of the buffer
	//buf.WriteString("\n")
	fmt.Fprintf(buf, "\n")

	// The Logger interface requires implementations to be safe for concurrent
	// use by multiple goroutines. For this implementation that means making
	// only one call to l.w.Write() for each call to Log.
	if _, err := l.w.Write(buf.Bytes()); err != nil {
		return err
	}
	return nil
}

type reformatData struct {
	time     time.Time
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
				data.time = t
			} else if t, err := time.Parse(time.RFC3339, getString(keyvals[i+1])); err == nil {
				data.time = t
			} else {
				// We failed to parse it this should be close enough
				data.time = time.Now()
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

	if data.level == "" {
		data.level = "info"
	}
	if data.time.IsZero() {
		data.time = time.Now()
	}

	return data
}

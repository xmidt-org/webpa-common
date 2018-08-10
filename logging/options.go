package logging

import (
	"io"
	"os"

	"github.com/go-kit/kit/log"
	"gopkg.in/natefinch/lumberjack.v2"
	"github.com/go-kit/kit/log/term"
)

const (
	StdoutFile = "stdout"
)

// Options stores the configuration of a Logger.  Lumberjack is used for rolling files.
type Options struct {
	// File is the system file path for the log file.  If set to "stdout", this will log to os.Stdout.
	// Otherwise, a lumberjack.Logger is created
	File string `json:"file"`

	// MaxSize is the lumberjack MaxSize
	MaxSize int `json:"maxsize"`

	// MaxAge is the lumberjack MaxAge
	MaxAge int `json:"maxage"`

	// MaxBackups is the lumberjack MaxBackups
	MaxBackups int `json:"maxbackups"`

	// JSON is a flag indicating whether JSON logging output is used.  The default is false,
	// meaning that logfmt output is used.
	JSON bool `json:"json"`

	// FMTType is to change the output style. The default is "term", for ease of use for debuging.
	// Another option is "fmt", for plain txt output
	FMTType string `json:"fmttype"`

	// Level is the error level to output: ERROR, INFO, WARN, or DEBUG.  Any unrecognized string,
	// including the empty string, is equivalent to passing ERROR.
	Level string `json:"level"`
}

func (o *Options) output() io.Writer {
	if o != nil && len(o.File) > 0 && o.File != StdoutFile {
		return &lumberjack.Logger{
			Filename:   o.File,
			MaxSize:    o.MaxSize,
			MaxAge:     o.MaxAge,
			MaxBackups: o.MaxBackups,
		}
	}

	return log.NewSyncWriter(os.Stdout)
}

func (o *Options) loggerFactory() func(io.Writer) log.Logger {
	if o != nil && o.JSON {
		return log.NewJSONLogger
	}

	if o != nil {
		switch o.FMTType {
		case "fmt":
			return log.NewLogfmtLogger
		case "term":
		default:
		}
	}
	return termLogger
}

func termLogger(writer io.Writer) log.Logger {
	colorFn := func(keyvals ...interface{}) term.FgBgColor {
		for i := 0; i < len(keyvals)-1; i += 2 {
			if keyvals[i] != "level" {
				continue
			}
			switch getString(keyvals[i+1]) {
			case "debug":
				return term.FgBgColor{Fg: term.DarkGray}
			case "info":
				return term.FgBgColor{Fg: term.Gray}
			case "warn":
				return term.FgBgColor{Fg: term.Yellow}
			case "error":
				return term.FgBgColor{Fg: term.Red}
			case "crit":
				return term.FgBgColor{Fg: term.Gray, Bg: term.DarkRed}
			default:
				return term.FgBgColor{}
			}
		}
		return term.FgBgColor{}
	}

	return term.NewColorLogger(writer, NewReformatLogger, colorFn)
}

type toString interface {
	String() string
}

func getString(obj interface{}) string {
	if s, ok := obj.(string); ok {
		return s
	}
	if levelObj, ok := obj.(toString); ok {
		return levelObj.String()
	}
	return ""
}

func (o *Options) level() string {
	if o != nil {
		return o.Level
	}

	return ""
}

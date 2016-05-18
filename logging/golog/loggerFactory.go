package golog

import (
	"github.com/Comcast/webpa-common/logging"
	"github.com/ian-kent/go-log/appenders"
	"github.com/ian-kent/go-log/levels"
	"github.com/ian-kent/go-log/logger"
	"os"
)

const (
	ConsoleFileName string  = "console"
	DefaultPattern  Pattern = "[%d] [%p] (%l) %m%n"
)

// adapter adapts the go-log Logger onto logging.Logger
type adapter struct {
	logger.Logger
}

func (a adapter) Printf(format string, parameters ...interface{}) {
	if !a.Enabled()[levels.INFO] {
		return
	}

	a.Appender().Write(
		levels.INFO,
		format,
		parameters...,
	)
}

// LoggerFactory is the golog-specific factory for logs.  It is configurable
// via JSON.
type LoggerFactory struct {
	File      string   `json:"file"`
	Level     LogLevel `json:"level"`
	Pattern   Pattern  `json:"pattern"`
	MaxSize   int64    `json:"maxSize"`
	MaxBackup int      `json:"maxBackup"`
}

var _ logging.LoggerFactory = (*LoggerFactory)(nil)

// NewAppender creates a golog Appender from this LoggerFactory's configuration
func (factory *LoggerFactory) NewAppender() (appenders.Appender, error) {
	var appender appenders.Appender
	if factory.File == ConsoleFileName {
		appender = appenders.Console()
	} else {
		if _, err := os.Stat(factory.File); err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			}

			if _, err = os.Create(factory.File); err != nil {
				return nil, err
			}
		}

		rollingFileAppender := appenders.RollingFile(factory.File, true)
		rollingFileAppender.MaxFileSize = factory.MaxSize
		rollingFileAppender.MaxBackupIndex = factory.MaxBackup
		appender = rollingFileAppender
	}

	if len(factory.Pattern) > 0 {
		appender.SetLayout(factory.Pattern.ToLayout())
	} else {
		appender.SetLayout(DefaultPattern.ToLayout())
	}

	return appender, nil
}

// NewLogger provides the implementation of logging.LoggerFactory
func (factory *LoggerFactory) NewLogger(name string) (logging.Logger, error) {
	if appender, err := factory.NewAppender(); err != nil {
		return nil, err
	} else {
		adapter := &adapter{logger.New(name)}
		adapter.SetLevel(levels.LogLevel(factory.Level))
		adapter.SetAppender(appender)

		return adapter, nil
	}
}

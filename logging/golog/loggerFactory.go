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

// LoggerFactory is the golog-specific factory for logs.  It is configurable
// via JSON.
type LoggerFactory struct {
	File      string   `json:"file"`
	Level     LogLevel `json:"level"`
	Name      string   `json:"name"`
	Pattern   Pattern  `json:"pattern"`
	MaxSize   int64    `json:"maxSize"`
	MaxBackup int      `json:"maxBackup"`
}

var _ logging.LoggerFactory = (*LoggerFactory)(nil)

// NewAppender creates a golog Appender from this LoggerFactory's configuration
func (factory *LoggerFactory) NewAppender() (appenders.Appender, error) {
	if factory.File == ConsoleFileName {
		return appenders.Console(), nil
	}

	if _, err := os.Stat(factory.File); os.IsNotExist(err) {
		if _, err = os.Create(factory.File); err != nil {
			return nil, err
		}
	}

	appender := appenders.RollingFile(factory.File, true)
	if len(factory.Pattern) > 0 {
		appender.SetLayout(factory.Pattern.ToLayout())
	} else {
		appender.SetLayout(DefaultPattern.ToLayout())
	}

	appender.MaxFileSize = factory.MaxSize
	appender.MaxBackupIndex = factory.MaxBackup
	return appender, nil
}

// NewLogger provides the implementation of logging.LoggerFactory
func (factory *LoggerFactory) NewLogger() (logging.Logger, error) {
	if appender, err := factory.NewAppender(); err != nil {
		return nil, err
	} else {
		gologger := logger.New(factory.Name)
		gologger.SetLevel(levels.LogLevel(factory.Level))
		gologger.SetAppender(appender)

		return gologger, nil
	}
}

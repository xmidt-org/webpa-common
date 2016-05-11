package golog

import (
	"github.com/ian-kent/go-log/appenders"
	"github.com/ian-kent/go-log/layout"
	"github.com/ian-kent/go-log/levels"
	"github.com/ian-kent/go-log/logger"
	"testing"
)

// TestCaller is designed to show that the %l pattern in ian-kent isn't working
func TestCaller(t *testing.T) {
	appender := appenders.Console()
	appender.SetLayout(layout.Pattern("%l %m%n"))
	gologger := logger.New("test")
	gologger.SetLevel(levels.DEBUG)
	gologger.SetAppender(appender)

	// run this test with -v to see this
	// the caller is clearly not this file
	gologger.Error("test")
}

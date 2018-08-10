package logging

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"bytes"
	"fmt"
	"time"
	"github.com/go-kit/kit/log"
)

func reformatLoggerSetup() (*bytes.Buffer, log.Logger) {
	buf := &bytes.Buffer{}
	return buf, NewReformatLogger(buf)
}

func TestReformatLogger(t *testing.T) {
	assert := assert.New(t)

	buf, logger := reformatLoggerSetup()

	err := logger.Log("msg", "hello", "level", "warn", "ts", time.Now(), "isCool", "YES")
	assert.Nil(err)

	//test
	expected := "WARN[00000] \thello\t\tisCool=YES \n"
	actual := buf.String()
	assert.Equal(expected, actual, fmt.Sprintf("want %#v, have %#v", expected, actual))
}

func TestReformatLoggerWithNoMSG(t *testing.T) {
	assert := assert.New(t)

	buf, logger := reformatLoggerSetup()

	err := logger.Log("level", "error", "key", "value")
	assert.Nil(err)

	expected := "ERRO[00000] key=value \n"
	actual := buf.String()
	assert.Equal(expected, actual, fmt.Sprintf("want %#v, have %#v", expected, actual))
}

func TestReformatLoggerWithNoLevel(t *testing.T) {
	assert := assert.New(t)

	buf, logger := reformatLoggerSetup()

	err := logger.Log("msg", "Calling Endpoint")
	assert.Nil(err)

	//test
	expected := "INFO[00000] \tCalling Endpoint\t\t\n"
	actual := buf.String()
	assert.Equal(expected, actual, fmt.Sprintf("want %#v, have %#v", expected, actual))
}

func TestReformatLoggerWithTime(t *testing.T) {
	assert := assert.New(t)

	buf, logger := reformatLoggerSetup()

	err := logger.Log("msg", "hello", "ts", time.Now().Add(time.Second*5))
	assert.Nil(err)


	expected := "INFO[00005] \thello\t\t\n"
	actual := buf.String()
	assert.Equal(expected, actual, fmt.Sprintf("want %#v, have %#v", expected, actual))
}

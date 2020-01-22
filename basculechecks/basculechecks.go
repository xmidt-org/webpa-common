package basculechecks

import (
	"context"
	"errors"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/goph/emperror"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/webpa-common/logging"
)

var (
	ErrNoVals                 = errors.New("expected at least one value")
	ErrNoAuth                 = errors.New("couldn't get request info: authorization not found")
	ErrNonstringVal           = errors.New("expected value to be a string")
	ErrNoValidCapabilityFound = errors.New("no valid capability for endpoint")
	ErrNoCapabilitiesAtKey    = errors.New("no capabilities found at key given")
	ErrCapabilitiesNotAList   = errors.New("capabilities aren't able to be converted to a list")
)

const (
	DefaultKey = "capabilities"
)

type capabilityCheck struct {
	prefixToMatch   *regexp.Regexp
	acceptAllMethod string
}

type capabilityLogger struct {
	check  *capabilityCheck
	logger log.Logger
}

var defaultLogger = log.NewNopLogger()

func (c *capabilityCheck) EnforceCapabilities(ctx context.Context, vals []interface{}) error {
	if len(vals) == 0 {
		return ErrNoVals
	}

	auth, ok := bascule.FromContext(ctx)
	if !ok {
		return ErrNoAuth
	}
	reqVal := auth.Request

	return c.CheckCapabilities(vals, reqVal)
}

func (c *capabilityLogger) OnAuthenticated(token bascule.Authentication) {
	if token.Authorization != "jwt" {
		c.logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), emperror.Wrap(errors.New("Authorization used isn't jwt"), "Cannot check endpoint against capabilities"))
		return
	}
	val, ok := token.Token.Attributes()[DefaultKey]
	if !ok {
		c.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "failed to get capabilities from token", logging.ErrorKey(), ErrNoCapabilitiesAtKey)
		return
	}
	capabilities, ok := val.([]interface{})
	if !ok {
		c.logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "failed to get list of capabilities", logging.ErrorKey(), ErrCapabilitiesNotAList)
		return
	}
	err := c.check.CheckCapabilities(capabilities, token.Request)
	if err != nil {
		log.With(c.logger, emperror.Context(err)...).
			Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "capability check failed", logging.ErrorKey(), err, "client id", token.Token.Principal)
	}
}

func NewCapabilityChecker(prefix string, acceptAllMethod string) (*capabilityCheck, error) {
	matchPrefix, err := regexp.Compile("^" + prefix + "(.+):(.+?)$")
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to compile prefix given", "prefix", prefix)
	}
	c := capabilityCheck{
		prefixToMatch:   matchPrefix,
		acceptAllMethod: acceptAllMethod,
	}
	return &c, nil
}

func NewCapabilityLogger(logger log.Logger, prefix string, acceptAllMethod string) (*capabilityLogger, error) {
	checker, err := NewCapabilityChecker(prefix, acceptAllMethod)
	if err != nil {
		return nil, emperror.Wrap(err, "failed to create capability checker")
	}
	c := capabilityLogger{
		check:  checker,
		logger: defaultLogger,
	}
	if logger != nil {
		c.logger = logger
	}
	return &c, nil
}

func (c *capabilityCheck) CheckCapabilities(capabilities []interface{}, requestInfo bascule.Request) error {
	urlToMatch := requestInfo.URL.EscapedPath()
	methodToMatch := requestInfo.Method
	for _, val := range capabilities {
		str, ok := val.(string)
		if !ok {
			return ErrNonstringVal
		}
		matches := c.prefixToMatch.FindStringSubmatch(str)
		if matches == nil || len(matches) < 3 {
			continue
		}

		method := matches[2]
		if method != c.acceptAllMethod && method != strings.ToLower(methodToMatch) {
			continue
		}

		re := regexp.MustCompile(matches[1]) //url regex that capability grants access to
		matchIdxs := re.FindStringIndex(requestInfo.URL.EscapedPath())
		if matchIdxs == nil {
			continue
		}
		if matchIdxs[0] == 0 {
			return nil
		}
	}
	return emperror.With(ErrNoValidCapabilityFound, "capabilitiesFound", capabilities, "urlToMatch", urlToMatch, "methodToMatch", methodToMatch)

}

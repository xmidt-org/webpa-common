package basculechecks

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/goph/emperror"
	"github.com/xmidt-org/bascule"
)

var (
	ErrNoVals                 = errors.New("expected at least one value")
	ErrNoAuth                 = errors.New("couldn't get request info: authorization not found")
	ErrNonstringVal           = errors.New("expected value to be a string")
	ErrNoValidCapabilityFound = errors.New("no valid capability for endpoint")
	ErrNilAttributes          = errors.New("nil attributes interface")
	ErrNilPrefix              = errors.New("prefix regular expression cannot be nil")
)

const (
	CapabilityKey = "capabilities"
	PartnerKey    = "allowedResources.allowedPartners"
)

type capabilityCheck struct {
	prefixToMatch *regexp.Regexp
	method        MethodConfig
}

type MethodConfig struct {
	AppendsCapability bool
	AcceptAllMethod   string
}

// CreateBasculeCheck creates a function that determines whether or not a
// client is authorized to make a request to an endpoint by comparing the
// method and url to the values at the CapabilityKey in the Attributes of a
// token.  The function created can error out or not based on the parameter
// passed, and the outcome of the check will be updated in a metric.
func (c *capabilityCheck) CreateBasculeCheck(errorOut bool) bascule.ValidatorFunc {
	return func(ctx context.Context, _ bascule.Token) error {
		auth, ok := bascule.FromContext(ctx)
		if !ok {
			if errorOut {
				return ErrNoAuth
			}
			return nil
		}

		_, err := c.Check(auth)
		if err != nil && errorOut {
			return err
		}

		return nil
	}
}

func (c *capabilityCheck) Check(auth bascule.Authentication) (string, error) {
	vals, reason, err := getCapabilities(auth.Token.Attributes())
	if err != nil {
		return reason, err
	}

	err = c.checkCapabilities(vals, auth.Request)
	if err != nil {
		return NoCapabilitiesMatch, err
	}
	return "", nil
}

// NewCapabilityChecker creates an object that produces a check on capabilities
// in bascule tokens, to be run by the bascule enforcer middleware.
func NewCapabilityChecker(prefix *regexp.Regexp, method MethodConfig) (*capabilityCheck, error) {
	if prefix == nil {
		return nil, ErrNilPrefix
	}

	c := capabilityCheck{
		prefixToMatch: prefix,
		method:        method,
	}
	return &c, nil
}

// NewCapabilityCheckerFromString creates an object that produces a check on capabilities
// in bascule tokens, to be run by the bascule enforcer middleware.
func NewCapabilityCheckerFromString(prefix string, method MethodConfig) (*capabilityCheck, error) {
	matchPrefix, err := regexp.Compile("^" + prefix + "(.+):(.+?)$")
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to compile prefix given", "prefix", prefix)
	}

	c := capabilityCheck{
		prefixToMatch: matchPrefix,
		method:        method,
	}
	return &c, nil
}

// checkCapabilities parses each capability to check it against the prefix
// expected, the url hit, and the method used.  If a match is found, no error is returned.
func (c *capabilityCheck) checkCapabilities(capabilities []string, requestInfo bascule.Request) error {
	urlToMatch := requestInfo.URL.EscapedPath()
	methodToMatch := requestInfo.Method
	for _, val := range capabilities {
		matches := c.prefixToMatch.FindStringSubmatch(val)
		if matches == nil || len(matches) < 3 {
			continue
		}

		method := matches[2]
		if method != c.method.AcceptAllMethod && method != strings.ToLower(methodToMatch) {
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

// getCapabilities runs some error checks while getting the list of
// capabilities from the attributes.
func getCapabilities(attributes bascule.Attributes) ([]string, string, error) {
	if attributes == nil {
		return []string{}, UndeterminedCapabilities, ErrNilAttributes
	}

	vals, ok := attributes.GetStringSlice(CapabilityKey)
	if !ok {
		return []string{}, UndeterminedCapabilities, fmt.Errorf("couldn't get capabilities using key %v", CapabilityKey)
	}

	if len(vals) == 0 {
		return []string{}, EmptyCapabilitiesList, ErrNoVals
	}

	return vals, "", nil

}

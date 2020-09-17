package basculechecks

import (
	"context"
	"errors"
	"fmt"

	"github.com/goph/emperror"
	"github.com/xmidt-org/bascule"
)

var (
	ErrNoVals                 = errors.New("expected at least one value")
	ErrNoAuth                 = errors.New("couldn't get request info: authorization not found")
	ErrNonstringVal           = errors.New("expected value to be a string")
	ErrNoValidCapabilityFound = errors.New("no valid capability for endpoint")
	ErrNilAttributes          = errors.New("nil attributes interface")
)

const (
	CapabilityKey = "capabilities"
	PartnerKey    = "allowedResources.allowedPartners"
)

type CapabilityChecker interface {
	Authorized(string, string, string) bool
}

type capabilitiesValidator struct {
	checker CapabilityChecker
}

// CreateBasculeCheck creates a function that determines whether or not a
// client is authorized to make a request to an endpoint by comparing the
// method and url to the values at the CapabilityKey in the Attributes of a
// token.  The function created can error out or not based on the parameter
// passed, and the outcome of the check will be updated in a metric.
func (c *capabilitiesValidator) CreateBasculeCheck(errorOut bool) bascule.ValidatorFunc {
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

func (c *capabilitiesValidator) Check(auth bascule.Authentication) (string, error) {
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

// NewCapabilitiesParser creates an object that produces a check on capabilities
// in bascule tokens, to be run by the bascule enforcer middleware.
func NewCapabilitiesParser(c CapabilityChecker) (*capabilitiesValidator, error) {
	if c == nil {
		return nil, ErrNilPrefix
	}

	checker := capabilitiesValidator{
		checker: c,
	}
	return &checker, nil
}

// checkCapabilities parses each capability to check it against the prefix
// expected, the url hit, and the method used.  If a match is found, no error is returned.
func (c *capabilitiesValidator) checkCapabilities(capabilities []string, requestInfo bascule.Request) error {
	urlToMatch := requestInfo.URL.EscapedPath()
	methodToMatch := requestInfo.Method
	for _, val := range capabilities {
		if c.checker.Authorized(val, urlToMatch, methodToMatch) {
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

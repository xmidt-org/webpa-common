package basculechecks

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log/level"

	"github.com/go-kit/kit/log"
	"github.com/goph/emperror"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/webpa-common/logging"
)

var (
	ErrNoVals                 = errors.New("expected at least one value")
	ErrNoAuth                 = errors.New("couldn't get request info: authorization not found")
	ErrNonstringVal           = errors.New("expected value to be a string")
	ErrNoValidCapabilityFound = errors.New("no valid capability for endpoint")
	ErrNilAttributes          = fmt.Errorf("nil attributes interface")
)

const (
	CapabilityKey = "capabilities"
	PartnerKey    = "allowedResources.allowedPartners"
)

type capabilityCheck struct {
	prefixToMatch   *regexp.Regexp
	endpoints       []*regexp.Regexp
	acceptAllMethod string
	measures        *AuthCapabilityCheckMeasures
}

var defaultLogger = log.NewNopLogger()

// CreateBasculeCheck creates a function that determines whether or not a
// client is authorized to make a request to an endpoint by comparing the
// method and url to the values at the CapabilityKey in the Attributes of a
// token.  The function created can error out or not based on the parameter
// passed, and the outcome of the check will be updated in a metric.
func (c *capabilityCheck) CreateBasculeCheck(errorOut bool) bascule.ValidatorFunc {
	return func(ctx context.Context, _ bascule.Token) error {
		// if we're not supposed to error out, the outcome should be accepted on failure
		failureOutcome := AcceptedOutcome
		if errorOut {
			// if we actually error out, the outcome is the request being rejected
			failureOutcome = RejectedOutcome
		}

		auth, ok := bascule.FromContext(ctx)
		if !ok {
			c.measures.CapabilityCheckOutcome.With(OutcomeLabel, failureOutcome, ReasonLabel, TokenMissing, ClientIDLabel, "", PartnerIDLabel, "", EndpointLabel, "").Add(1)
			if errorOut {
				return ErrNoAuth
			}
			return nil
		}

		client, partnerID, endpoint, reason, err := c.prepMetrics(auth)
		labels := []string{ClientIDLabel, client, PartnerIDLabel, partnerID, EndpointLabel, endpoint}
		if err != nil {
			labels = append(labels, OutcomeLabel, failureOutcome, ReasonLabel, reason)
			c.measures.CapabilityCheckOutcome.With(labels...).Add(1)
			if errorOut {
				return err
			}
			return nil
		}

		vals, reason, err := getCapabilities(auth.Token.Attributes())
		if err != nil {
			labels = append(labels, OutcomeLabel, failureOutcome, ReasonLabel, reason)
			c.measures.CapabilityCheckOutcome.With(labels...).Add(1)
			if errorOut {
				return err
			}
			return nil
		}

		err = c.checkCapabilities(vals, auth.Request)
		if err != nil {
			labels = append(labels, OutcomeLabel, failureOutcome, ReasonLabel, NoCapabilitiesMatch)
			c.measures.CapabilityCheckOutcome.With(labels...).Add(1)
			if errorOut {
				return err
			}
			return nil
		}

		labels = append(labels, OutcomeLabel, AcceptedOutcome, ReasonLabel, "")
		c.measures.CapabilityCheckOutcome.With(labels...).Add(1)
		return nil
	}
}

// NewCapabilityChecker creates an object that produces a check on capabilities
// in bascule tokens, to be run by the bascule enforcer middleware.
func NewCapabilityChecker(m *AuthCapabilityCheckMeasures, prefix string, acceptAllMethod string, endpoints []*regexp.Regexp) (*capabilityCheck, error) {
	if m == nil {
		return nil, errors.New("nil capability check measures")
	}
	matchPrefix, err := regexp.Compile("^" + prefix + "(.+):(.+?)$")
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to compile prefix given", "prefix", prefix)
	}

	c := capabilityCheck{
		prefixToMatch:   matchPrefix,
		endpoints:       endpoints,
		acceptAllMethod: acceptAllMethod,
		measures:        m,
	}
	return &c, nil
}

// NewCapabilityCheckerFromStrings creates the capability checker, and allows
// consumers to provide a list of string endpoints, rather than regular
// expressions.
func NewCapabilityCheckerFromStrings(m *AuthCapabilityCheckMeasures, prefix string, acceptAllMethod string, endpoints []string, logger log.Logger) (*capabilityCheck, error) {
	var endpointRegexps []*regexp.Regexp
	l := logger
	if logger == nil {
		l = defaultLogger
	}
	for _, e := range endpoints {
		r, err := regexp.Compile(e)
		if err != nil {
			l.Log(level.Key(), level.WarnValue(), logging.MessageKey(), "failed to compile regular expression", "regex", e, logging.ErrorKey(), err.Error())
			continue
		}
		endpointRegexps = append(endpointRegexps, r)
	}

	return NewCapabilityChecker(m, prefix, acceptAllMethod, endpointRegexps)

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

// prepMetrics gathers the information needed for metric label information.
func (c *capabilityCheck) prepMetrics(auth bascule.Authentication) (string, string, string, string, error) {
	// getting metric information
	client := auth.Token.Principal()
	partnerIDs, ok := auth.Token.Attributes().GetStringSlice(PartnerKey)
	if !ok {
		return client, "", "", UndeterminedPartnerID, fmt.Errorf("couldn't get partner IDs from attributes using key %v", PartnerKey)
	}
	partnerID := determinePartnerMetric(partnerIDs)
	escapedURL := auth.Request.URL.EscapedPath()
	endpoint := determineEndpointMetric(c.endpoints, escapedURL)
	return client, partnerID, endpoint, "", nil

}

// determinePartnerMetric takes a list of partners and decides what the partner
// metric label should be.
func determinePartnerMetric(partners []string) string {
	if len(partners) < 1 {
		return "none"
	}
	if len(partners) == 1 {
		if partners[0] == "*" {
			return "wildcard"
		}
		return partners[0]
	}
	for _, partner := range partners {
		if partner == "*" {
			return "wildcard"
		}
	}
	return "many"

}

// determineEndpointMetric takes a list of regular expressions and applies them
// to the url of the request to decide what the endpoint metric label should be.
func determineEndpointMetric(endpoints []*regexp.Regexp, urlHit string) string {
	for _, r := range endpoints {
		idxs := r.FindStringIndex(urlHit)
		if idxs == nil {
			continue
		}
		if idxs[0] == 0 {
			return r.String()
		}
	}
	return "not_recognized"
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

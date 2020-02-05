package basculechecks

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/goph/emperror"
	"github.com/xmidt-org/bascule"
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
	DefaultRegex  = `(mac|uuid|dns|serial):([^/]+)`
)

type capabilityCheck struct {
	prefixToMatch   *regexp.Regexp
	endpointToMatch *regexp.Regexp
	acceptAllMethod string
	measures        *AuthCapabilityCheckMeasures
}

var defaultLogger = log.NewNopLogger()

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

func NewCapabilityChecker(m *AuthCapabilityCheckMeasures, prefix string, acceptAllMethod string) (*capabilityCheck, error) {
	if m == nil {
		return nil, errors.New("nil capability check measures")
	}
	matchPrefix, err := regexp.Compile("^" + prefix + "(.+):(.+?)$")
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to compile prefix given", "prefix", prefix)
	}
	matchEndpoint, err := regexp.Compile(DefaultRegex)
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to compile endpoint regex given", "endpoint", DefaultRegex)
	}

	c := capabilityCheck{
		prefixToMatch:   matchPrefix,
		endpointToMatch: matchEndpoint,
		acceptAllMethod: acceptAllMethod,
		measures:        m,
	}
	return &c, nil
}

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

func (c *capabilityCheck) prepMetrics(auth bascule.Authentication) (string, string, string, string, error) {
	// getting metric information
	client := auth.Token.Principal()
	partnerIDs, ok := auth.Token.Attributes().GetStringSlice(PartnerKey)
	if !ok {
		return client, "", "", UndeterminedPartnerID, fmt.Errorf("couldn't get partner IDs from attributes using key %v", PartnerKey)
	}
	partnerID := determinePartnerMetric(partnerIDs)
	escapedURL := auth.Request.URL.EscapedPath()
	i := c.endpointToMatch.FindStringIndex(escapedURL)
	endpoint := escapedURL
	if i != nil {
		endpoint = escapedURL[:i[0]] + "..." + escapedURL[i[1]:]
	}
	return client, partnerID, endpoint, "", nil

}

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

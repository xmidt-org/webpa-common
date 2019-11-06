package basculechecks

import (
	"context"
	"errors"
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
)

func CreateValidCapabilityCheck(prefix string, acceptAllMethod string) (func(context.Context, []interface{}) error, error) {
	matchPrefix, err := regexp.Compile("^" + prefix + "(.+):(.+?)$")
	if err != nil {
		return nil, emperror.WrapWith(err, "failed to compile prefix given", "prefix", prefix)
	}
	return func(ctx context.Context, vals []interface{}) error {
		if len(vals) == 0 {
			return ErrNoVals
		}

		auth, ok := bascule.FromContext(ctx)
		if !ok {
			return ErrNoAuth
		}
		reqVal := auth.Request

		for _, val := range vals {
			str, ok := val.(string)
			if !ok {
				return ErrNonstringVal
			}
			matches := matchPrefix.FindStringSubmatch(str)
			if matches == nil || len(matches) < 3 {
				continue
			}

			method := matches[2]
			if method != acceptAllMethod && method != strings.ToLower(reqVal.Method) {
				continue
			}

			re := regexp.MustCompile(matches[1])
			matchIdxs := re.FindStringIndex(reqVal.URL)
			if matchIdxs == nil {
				continue
			}
			if matchIdxs[0] == 0 {
				return nil
			}
		}
		return ErrNoValidCapabilityFound
	}, nil
}

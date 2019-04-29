package basculechecks

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/Comcast/comcast-bascule/bascule"
)

type CapabilityConfig struct {
	FirstPiece      string
	SecondPiece     string
	ThirdPiece      string
	AcceptAllMethod string
}

func CreateValidCapabilityCheck(config CapabilityConfig) func(context.Context, []interface{}) error {
	return func(ctx context.Context, vals []interface{}) error {
		if len(vals) == 0 {
			return errors.New("expected at least one value")
		}

		auth, ok := bascule.FromContext(ctx)
		if !ok {
			return errors.New("couldn't get request info")
		}
		reqVal := auth.Request

		for _, val := range vals {
			str, ok := val.(string)
			if !ok {
				return errors.New("expected value to be a string")
			}
			if len(str) == 0 {
				return errors.New("expected string to be nonempty")
			}
			pieces := strings.Split(str, ":")
			if len(pieces) != 5 {
				return fmt.Errorf("malformed string: [%v]", str)
			}
			method := pieces[4]
			if method != config.AcceptAllMethod && method != strings.ToLower(reqVal.Method) {
				continue
			}
			if pieces[0] != config.FirstPiece || pieces[1] != config.SecondPiece || pieces[2] != config.ThirdPiece {
				continue
			}
			matched, err := regexp.MatchString(pieces[3], reqVal.URL)
			if err != nil {
				continue
			}
			if matched {
				return nil
			}
		}
		return errors.New("no valid capability for endpoint")
	}
}

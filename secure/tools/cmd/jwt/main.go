package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
)

var (
	ErrorMalformedToken = errors.New("That token is not valid")
)

type Arguments struct {
	Token  string
	KeyURI string
}

func decodeAndUnmarshal(encoding *base64.Encoding, encoded []byte) (map[string]interface{}, error) {
	decoder := base64.NewDecoder(encoding, bytes.NewReader(encoded))
	decoded, err := ioutil.ReadAll(decoder)
	if err != nil {
		return nil, err
	}

	unmarshalled := make(map[string]interface{})
	err = json.Unmarshal(decoded, &unmarshalled)
	return unmarshalled, err
}

func displayToken(token []byte) error {
	parts := bytes.Split(token, []byte{'.'})
	if len(parts) < 2 {
		return ErrorMalformedToken
	}

	header, err := decodeAndUnmarshal(base64.StdEncoding, parts[0])
	if err != nil {
		return err
	}

	fmt.Println(header)

	payload, err := decodeAndUnmarshal(base64.StdEncoding, parts[1])
	if err != nil {
		return err
	}

	fmt.Println(payload)
	return nil
}

func main() {
	var arguments Arguments
	flag.StringVar(&arguments.Token, "t", "", "The JWT token.  If not supplied, a token is expected from stdin.")
	flag.StringVar(&arguments.KeyURI, "k", "", "the URI of a public key for verification (optional)")
	flag.Parse()

	var (
		token []byte
		err   error
	)

	if len(arguments.Token) == 0 {
		token, err = ioutil.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stdout, "Unable to read token from stdin: %s\n", err)
			return
		}
	} else {
		token = []byte(arguments.Token)
	}

	if err = displayToken(token); err != nil {
		fmt.Fprintf(os.Stdout, "%s\n", err)
	}
}

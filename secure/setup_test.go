package secure

import (
	"fmt"
	"github.com/SermoDigital/jose/jws"
	"os"
	"testing"
)

const (
	publicKeyFileName  = "jwt-key.pub"
	privateKeyFileName = "jwt-key"
)

var (
	publicKeyFileURI  string
	privateKeyFileURI string
)

func TestMain(m *testing.M) {
	currentDirectory, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to obtain current working directory: %v\n", err)
		return
	}
}

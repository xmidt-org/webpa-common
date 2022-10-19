package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
)

type RouteBuilder struct {
	Issuer   string
	Logger   *log.Logger
	KeyStore *KeyStore
}

func (rb RouteBuilder) Build(router *mux.Router) {
	keyHandler := KeyHandler{
		BasicHandler{
			keyStore: rb.KeyStore,
			logger:   rb.Logger,
		},
	}

	keysRouter := router.Methods("GET").Subrouter()

	keysRouter.HandleFunc("/keys", keyHandler.ListKeys)
	rb.Logger.Println("GET /keys returns a list of the identifiers of available keys")

	keysRouter.HandleFunc(fmt.Sprintf("/keys/{%s}", KeyIDVariableName), keyHandler.GetKey)
	rb.Logger.Println("GET /keys/{kid} returns the public key associated with the given key identifier.  There is no way to look up the associated private key.")

	issueHandler := IssueHandler{
		BasicHandler: BasicHandler{
			keyStore: rb.KeyStore,
			logger:   rb.Logger,
		},
		decoder: schema.NewDecoder(),
		issuer:  rb.Issuer,
	}

	issueRouter := router.
		Path("/jws").
		Queries(KeyIDVariableName, "").
		Subrouter()

	issueRouter.Methods("GET").
		HandlerFunc(issueHandler.SimpleIssue)
	rb.Logger.Println("GET /jws?kid={kid} generates a JWT signed with the associated private key.  Additional URL parameters are interpreted as reserved claims, e.g. exp")

	issueRouter.Methods("PUT", "POST").
		Headers("Content-Type", "application/json").
		HandlerFunc(issueHandler.IssueUsingBody)
	rb.Logger.Println("PUT/POST /jws generates a JWT signed with the associated private key.  Additional URL parmaeters are interpreted as reserved claims, e.g. exp")
}

func main() {
	logger := log.New(os.Stdout, "[INFO]  ", log.LstdFlags|log.LUTC)

	var configurationFileName string
	flag.StringVar(&configurationFileName, "f", "", "the required configuration file")
	flag.Parse()

	configuration, err := ParseConfiguration(configurationFileName)
	if err != nil {
		logger.Fatalf("Unable to parse configuration file: %s\n", err)
	}

	keyStore, err := NewKeyStore(logger, configuration)
	if err != nil {
		logger.Fatalf("Unable to initialize key store: %s\n", err)
	}

	logger.Printf("Initialized key store with %d keys: %s\n", keyStore.Len(), keyStore.KeyIDs())

	issuer := configuration.Issuer
	if len(issuer) == 0 {
		issuer = DefaultIssuer
	}

	router := mux.NewRouter()
	RouteBuilder{
		Issuer:   issuer,
		Logger:   logger,
		KeyStore: keyStore,
	}.Build(router)

	bindAddress := configuration.BindAddress
	if len(bindAddress) == 0 {
		bindAddress = DefaultBindAddress
	}

	server := &http.Server{
		Addr:     bindAddress,
		Handler:  router,
		ErrorLog: logger,
	}

	logger.Printf("Listening on %s\n", bindAddress)
	log.Fatalln(server.ListenAndServe())
}

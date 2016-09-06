package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"log"
	"net/http"
	"os"
)

func addRoutes(issuer string, errorLogger *log.Logger, keyStore *KeyStore, router *mux.Router) {
	keyHandler := KeyHandler{
		BasicHandler{
			keyStore:    keyStore,
			errorLogger: errorLogger,
		},
	}

	keysRouter := router.Methods("GET").Subrouter()
	keysRouter.HandleFunc("/keys", keyHandler.ListKeys)
	keysRouter.HandleFunc(fmt.Sprintf("/keys/{%s}", KeyIDVariableName), keyHandler.GetKey)

	issueHandler := IssueHandler{
		BasicHandler: BasicHandler{
			keyStore:    keyStore,
			errorLogger: errorLogger,
		},
		decoder: schema.NewDecoder(),
		issuer:  issuer,
	}

	issueRouter := router.
		Path("/jws").
		Queries(KeyIDVariableName, "").
		Subrouter()

	issueRouter.Methods("GET").
		HandlerFunc(issueHandler.SimpleIssue)

	issueRouter.Methods("PUT", "POST").
		Headers("Content-Type", "application/json").
		HandlerFunc(issueHandler.IssueUsingBody)
}

func main() {
	infoLogger := log.New(os.Stdout, "[INFO]  ", log.LstdFlags|log.LUTC)
	errorLogger := log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.LUTC)

	var configurationFileName string
	flag.StringVar(&configurationFileName, "f", "", "the required configuration file")
	flag.Parse()

	configuration, err := ParseConfiguration(configurationFileName)
	if err != nil {
		errorLogger.Fatalf("Unable to parse configuration file: %s\n", err)
	}

	keyStore, err := NewKeyStore(infoLogger, configuration)
	if err != nil {
		errorLogger.Fatalf("Unable to initialize key store: %s\n", err)
	}

	infoLogger.Printf("Initialized key store with %d keys: %s\n", keyStore.Len(), keyStore.KeyIDs())

	issuer := configuration.Issuer
	if len(issuer) == 0 {
		issuer = DefaultIssuer
	}

	router := mux.NewRouter()
	addRoutes(issuer, errorLogger, keyStore, router)

	bindAddress := configuration.BindAddress
	if len(bindAddress) == 0 {
		bindAddress = DefaultBindAddress
	}

	server := &http.Server{
		Addr:     bindAddress,
		Handler:  router,
		ErrorLog: errorLogger,
	}

	infoLogger.Printf("Listening on %s\n", bindAddress)
	log.Fatalln(server.ListenAndServe())
}

package main

import (
	"flag"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
)

func addRoutes(errorLogger *log.Logger, keyStore *KeyStore, router *mux.Router) {
	router.Handle(
		fmt.Sprintf("/keys/{%s}", KeyIdVariableName),
		&KeyHandler{
			keyStore:    keyStore,
			errorLogger: errorLogger,
		},
	)

	router.Handle(
		"/keys",
		&ListKeysHandler{
			keyStore:    keyStore,
			errorLogger: errorLogger,
		},
	)
}

func main() {
	mainLogger := log.New(os.Stdout, "[INFO]  ", log.LstdFlags|log.LUTC)
	errorLogger := log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.LUTC)

	var configurationFileName string
	flag.StringVar(&configurationFileName, "f", "", "the required configuration file")
	flag.Parse()

	configuration, err := ParseConfiguration(configurationFileName)
	if err != nil {
		errorLogger.Fatalf("Unable to parse configuration file: %s\n", err)
	}

	keyStore, err := NewKeyStore(configuration)
	if err != nil {
		errorLogger.Fatalf("Unable to initialize key store: %s\n", err)
	}

	router := mux.NewRouter()
	addRoutes(errorLogger, keyStore, router)

	bindAddress := configuration.BindAddress
	if len(bindAddress) == 0 {
		bindAddress = DefaultBindAddress
	}

	server := &http.Server{
		Addr:     bindAddress,
		Handler:  router,
		ErrorLog: errorLogger,
	}

	mainLogger.Printf("Listening on %s\n", bindAddress)
	log.Fatalln(server.ListenAndServe())
}

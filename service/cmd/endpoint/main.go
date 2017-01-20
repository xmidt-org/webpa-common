package main

import (
	"fmt"
	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/server"
	"github.com/Comcast/webpa-common/service"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
)

const (
	applicationName = "endpoint"
)

func newFlagSet() *pflag.FlagSet {
	flagSet := pflag.NewFlagSet(applicationName, pflag.ExitOnError)
	flagSet.String("connection", service.DefaultServer, "the zookeeper connection string")
	flagSet.String("serviceName", service.DefaultServiceName, "the service name this endpoint will register with")
	return flagSet
}

func endpoint(arguments []string) int {
	var (
		logger = logging.DefaultLogger()

		f = newFlagSet()
		v = viper.New()
	)

	if err := server.Configure(applicationName, arguments, f, v); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	_, registrar, err := service.New(logger, nil, v.Sub(service.DiscoveryKey))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	watch, err := registrar.Watch()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		return 1
	}

	service.Subscribe(logger, watch, func(update []string) {
		fmt.Printf("Updated endpoints: %v\n", update)
	})

	fmt.Println("Send any signal to this process to exit ...")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals)
	<-signals

	return 0
}

func main() {
	os.Exit(endpoint(os.Args))
}

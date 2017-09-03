package main

import (
	"fmt"
	"os"
	"os/signal"

	"github.com/Comcast/webpa-common/logging"
	"github.com/Comcast/webpa-common/server"
	"github.com/Comcast/webpa-common/service"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/sd"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
		logger = logging.New(&logging.Options{JSON: true, Level: "debug"})

		f = newFlagSet()
		v = viper.New()
	)

	if err := server.Configure(applicationName, arguments, f, v); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not configure Viper", logging.ErrorKey(), err)
		return 1
	}

	if err := v.ReadInConfig(); err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not read Viper configuration", logging.ErrorKey(), err)
		return 1
	}

	serviceOptions, err := service.FromViper(service.Sub(v))
	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not read service options", logging.ErrorKey(), err)
		return 1
	}

	logger.Log(level.Key(), level.InfoValue(), "serviceOptions", serviceOptions)
	services, err := service.New(serviceOptions)
	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not initialize service discovery", logging.ErrorKey(), err)
		return 1
	}

	// Register will only register something if the Registration field is set on the options
	defer services.Deregister()
	services.Register()

	instancer, err := services.NewInstancer()
	if err != nil {
		logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "Could not create instancer", logging.ErrorKey(), err)
		return 1
	}

	go func() {
		events := make(chan sd.Event, 10)
		defer instancer.Deregister(events)
		instancer.Register(events)

		for {
			select {
			case e := <-events:
				if e.Err != nil {
					logger.Log(level.Key(), level.ErrorValue(), logging.MessageKey(), "service discovery error", logging.ErrorKey(), e.Err)
				} else {
					logger.Log(level.Key(), level.InfoValue(), logging.MessageKey(), "updated instances", "instances", e.Instances)
				}
			}
		}
	}()

	fmt.Println("Send any signal to this process to exit ...")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals)
	<-signals

	return 0
}

func main() {
	os.Exit(endpoint(os.Args))
}

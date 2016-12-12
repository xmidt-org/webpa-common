package main

import (
	"fmt"
	"github.com/Comcast/webpa-common/server"
	"github.com/Comcast/webpa-common/service"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"os/signal"
)

func newFlagSet(name string) *pflag.FlagSet {
	flagSet := pflag.NewFlagSet(name, pflag.ExitOnError)
	flagSet.String("connection", service.DefaultServer, "the zookeeper connection string")
	flagSet.String("serviceName", service.DefaultServiceName, "the service name this endpoint will register with")
	return flagSet
}

func main() {
	flagSet := newFlagSet("endpoint")
	viper := viper.New()
	if err := server.ReadInConfig("endpoint", viper, flagSet, nil); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}

	options := new(service.Options)
	if err := viper.Unmarshal(options); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to unmarshal options: %s\n", err)
		os.Exit(2)
	}

	fmt.Printf("Unmarshalled options: %#v\n", options)
	registrar := service.NewRegistrarWatcher(options)
	if _, err := service.RegisterAll(registrar, options); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to register endpoints: %s\n", err)
		os.Exit(2)
	}

	if watch, err := registrar.Watch(); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to set watch: %s\n", err)
	} else {
		service.Subscribe(watch, nil, func(update []string) {
			fmt.Printf("Updated endpoints: %v\n", update)
		})
	}

	fmt.Println("Send any signal to this process to exit ...")
	signals := make(chan os.Signal, 1)
	signal.Notify(signals)
	<-signals
}

package main

import (
	"fmt"
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

func newViper(flagSet *pflag.FlagSet) *viper.Viper {
	viper := viper.New()
	viper.SetConfigName("endpoint")

	viper.AddConfigPath("/etc/endpoint/")
	viper.AddConfigPath("$HOME/.endpoint")
	viper.AddConfigPath("./")

	viper.SetEnvPrefix("endpoint")
	viper.AutomaticEnv()

	viper.BindPFlags(flagSet)

	return viper
}

func main() {
	flagSet := newFlagSet("endpoint")
	viper := newViper(flagSet)
	flagSet.Parse(os.Args)
	if err := viper.ReadInConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read viper configuration: %s\n", err)
	} else {
		fmt.Printf("Using configuration file: %s\n", viper.ConfigFileUsed())
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

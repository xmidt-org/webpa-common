// SPDX-FileCopyrightText: 2025 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xmidt-org/sallust"
	"github.com/xmidt-org/webpa-common/v2/xmetrics"
	"go.uber.org/zap"
)

const (
	// DefaultPrimaryAddress is the bind address of the primary server (e.g. talaria, petasos, etc)
	DefaultPrimaryAddress = ":8080"

	// DefaultHealthAddress is the bind address of the health check server
	DefaultHealthAddress = ":8081"

	// DefaultMetricsAddress is the bind address of the metrics server
	DefaultMetricsAddress = ":8082"

	// DefaultPprofAddress is the bind address of the pprof server
	DefaultPprofAddress = ":6060"

	// DefaultHealthLogInterval is the interval at which health statistics are emitted
	// when a non-positive log interval is specified
	DefaultHealthLogInterval time.Duration = time.Duration(60 * time.Second)

	// DefaultLogConnectionState is the default setting for logging connection state messages.  This
	// value is primarily used when a *WebPA value is nil.
	DefaultLogConnectionState = false

	// AlternateSuffix is the suffix appended to the server name, along with a period (.), for
	// logging information pertinent to the alternate server.
	AlternateSuffix = "alternate"

	// DefaultProject is used as a metrics namespace when one is not defined.
	DefaultProject = "xmidt"

	// HealthSuffix is the suffix appended to the server name, along with a period (.), for
	// logging information pertinent to the health server.
	HealthSuffix = "health"

	// PprofSuffix is the suffix appended to the server name, along with a period (.), for
	// logging information pertinent to the pprof server.
	PprofSuffix = "pprof"

	// MetricsSuffix is the suffix appended to the server name, along with a period (.), for
	// logging information pertinent to the metrics server.
	MetricsSuffix = "metrics"

	// FileFlagName is the name of the command-line flag for specifying an alternate
	// configuration file for Viper to hunt for.
	FileFlagName = "file"

	// FileFlagShorthand is the command-line shortcut flag for FileFlagName
	FileFlagShorthand = "f"

	// CPUProfileFlagName is the command-line flag for creating a cpuprofile on the server
	CPUProfileFlagName = "cpuprofile"

	// CPUProfileShortHand is the command-line shortcut for creating cpushorthand on the server
	CPUProfileShorthand = "c"

	// MemProfileFlagName is the command-line flag for creating memprofile on the server
	MemProfileFlagName = "memprofile"

	// MemProfileShortHand is the command-line shortcut for creating memprofile on the server
	MemProfileShorthand = "m"
)

// ConfigureFlagSet adds the standard set of WebPA flags to the supplied FlagSet.  Use of this function
// is optional, and necessary only if the standard flags should be supported.  However, this is highly desirable,
// as ConfigureViper can make use of the standard flags to tailor how configuration is loaded or if gathering cpuprofile
// or memprofile data is needed.
func ConfigureFlagSet(applicationName string, f *pflag.FlagSet) {
	f.StringP(FileFlagName, FileFlagShorthand, applicationName, "base name of the configuration file")
	f.StringP(CPUProfileFlagName, CPUProfileShorthand, "cpuprofile", "base name of the cpuprofile file")
	f.StringP(MemProfileFlagName, MemProfileShorthand, "memprofile", "base name of the memprofile file")
}

// create CPUProfileFiles creates a cpu profile of the server, its triggered by the optional flag cpuprofile
//
// the CPU profile is created on the server's start
func CreateCPUProfileFile(v *viper.Viper, fp *pflag.FlagSet, l *zap.Logger) {
	if fp == nil {
		return
	}

	flag := fp.Lookup("cpuprofile")
	if flag == nil {
		return
	}

	f, err := os.Create(flag.Value.String())
	if err != nil {
		l.Info(fmt.Sprintf("could not create CPU profile: %v", err))
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		l.Info(fmt.Sprintf("could not start CPU profile: %v", err))
	}

	defer pprof.StopCPUProfile()
}

// Create CPUProfileFiles creates a memory profile of the server, its triggered by the optional flag memprofile
//
// the memory profile is created on the server's exit.
// this function should be used within the application.
func CreateMemoryProfileFile(_ *viper.Viper, fp *pflag.FlagSet, l *zap.Logger) {
	if fp == nil {
		return
	}

	flag := fp.Lookup("memprofile")
	if flag == nil {
		return
	}

	f, err := os.Create(flag.Value.String())
	if err != nil {
		l.Info(fmt.Sprintf("could not create memory profile: %v", err))
	}

	defer f.Close()
	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		l.Info(fmt.Sprintf("could not write memory profile: %v", err))
	}
}

// ConfigureViper configures a Viper instances using the opinionated WebPA settings.  All WebPA servers should
// use this function.
//
// The flagSet is optional.  If supplied, it will be bound to the given Viper instance.  Additionally, if the
// flagSet has a FileFlagName flag, it will be used as the configuration name to hunt for instead of the
// application name.
func ConfigureViper(applicationName string, f *pflag.FlagSet, v *viper.Viper) (err error) {
	v.AddConfigPath(fmt.Sprintf("/etc/%s", applicationName))
	v.AddConfigPath(fmt.Sprintf("$HOME/.%s", applicationName))
	v.AddConfigPath(".")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetEnvPrefix(applicationName)
	v.AutomaticEnv()

	v.SetDefault("primary.name", applicationName)
	v.SetDefault("primary.address", DefaultPrimaryAddress)
	v.SetDefault("primary.logConnectionState", DefaultLogConnectionState)

	v.SetDefault("alternate.name", fmt.Sprintf("%s.%s", applicationName, AlternateSuffix))

	v.SetDefault("health.name", fmt.Sprintf("%s.%s", applicationName, HealthSuffix))
	v.SetDefault("health.address", DefaultHealthAddress)
	v.SetDefault("health.logInterval", DefaultHealthLogInterval)
	v.SetDefault("health.logConnectionState", DefaultLogConnectionState)

	v.SetDefault("pprof.name", fmt.Sprintf("%s.%s", applicationName, PprofSuffix))
	v.SetDefault("pprof.address", DefaultPprofAddress)
	v.SetDefault("pprof.logConnectionState", DefaultLogConnectionState)

	v.SetDefault("metric.name", fmt.Sprintf("%s.%s", applicationName, MetricsSuffix))
	v.SetDefault("metric.address", DefaultMetricsAddress)

	v.SetDefault("project", DefaultProject)

	configName := applicationName
	if f != nil {
		if fileFlag := f.Lookup(FileFlagName); fileFlag != nil {
			// use the command-line to specify the base name of the file to be searched for
			configName = fileFlag.Value.String()
		}

		err = v.BindPFlags(f)
	}

	v.SetConfigName(configName)
	return
}

/*
Configure is a one-stop shopping function for preparing WebPA configuration.  This function
does not itself read in configuration from the Viper environment.  Typical usage is:

	var (
	  f = pflag.NewFlagSet()
	  v = viper.New()
	)

	if err := server.Configure("petasos", os.Args, f, v); err != nil {
	  // deal with the error, possibly just exiting
	}

	// further customizations to the Viper instance can be done here

	if err := v.ReadInConfig(); err != nil {
	  // more error handling
	}

Usage of this function is only necessary if custom configuration is needed.  Normally,
using New will suffice.
*/
func Configure(applicationName string, arguments []string, f *pflag.FlagSet, v *viper.Viper) (err error) {
	if f != nil {
		ConfigureFlagSet(applicationName, f)
		err = f.Parse(arguments)
		if err != nil {
			return
		}
	}

	err = ConfigureViper(applicationName, f, v)
	return
}

/*
Initialize handles the bootstrapping of the server code for a WebPA node.  It configures Viper,
reads configuration, and unmarshals the appropriate objects.  This function is typically all that's
needed to fully instantiate a WebPA server.  Typical usage:

	var (
	  f = pflag.NewFlagSet()
	  v = viper.New()

	  // can customize both the FlagSet and the Viper before invoking New
	  logger, registry, webPA, err = server.Initialize("petasos", os.Args, f, v)
	)

	if err != nil {
	  // deal with the error, possibly just exiting
	}

Note that the FlagSet is optional but highly encouraged.  If not supplied, then no command-line binding
is done for the unmarshalled configuration.

This function always returns a logger, regardless of any errors.  This allows clients to use the returned
logger when reporting errors.  This function falls back to a logger that writes to os.Stdout if it cannot
create a logger from the Viper environment.
*/
func Initialize(applicationName string, arguments []string, f *pflag.FlagSet, v *viper.Viper, modules ...xmetrics.Module) (logger *zap.Logger, registry xmetrics.Registry, webPA *WebPA, err error) {
	defer func() {
		if err != nil {
			// never return a WebPA in the presence of an error, to
			// avoid an ambiguous API
			webPA = nil

			// Make sure there's at least a default logger for the caller to use
			logger = zap.Must(zap.NewProductionConfig().Build())
		}
	}()

	if err = Configure(applicationName, arguments, f, v); err != nil {
		return
	}

	if err = v.ReadInConfig(); err != nil {
		return
	}

	webPA = &WebPA{
		ApplicationName: applicationName,
	}

	err = v.Unmarshal(webPA)
	if err != nil {
		return
	}

	var (
		zConfig sallust.Config
	)
	// Get touchstone & zap configurations
	v.UnmarshalKey("zap", &zConfig)
	logger = zap.Must(zConfig.Build())

	logger.Info("initialized Viper environment", zap.String("configurationFile", v.ConfigFileUsed()))

	if len(webPA.Metric.MetricsOptions.Namespace) == 0 {
		webPA.Metric.MetricsOptions.Namespace = applicationName
	}

	if len(webPA.Metric.MetricsOptions.Subsystem) == 0 {
		webPA.Metric.MetricsOptions.Subsystem = applicationName
	}

	webPA.Metric.MetricsOptions.Logger = logger
	registry, err = webPA.Metric.NewRegistry(modules...)
	if err != nil {
		return
	}

	CreateCPUProfileFile(v, f, logger)

	return
}

package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"metrics-collector/cf"
	"metrics-collector/config"
	"metrics-collector/server"
	"os"

	"github.com/cloudfoundry/noaa/consumer"
	"github.com/pivotal-golang/lager"
)

func main() {
	var path string
	flag.StringVar(&path, "c", "", "config file")
	flag.Parse()
	if path == "" {
		fmt.Fprintln(os.Stderr, "missing config file")
		os.Exit(1)
	}

	var conf *config.Config
	var err error
	conf, err = config.LoadConfigFromFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read config from file '%s' : %s\n", path, err.Error())
		os.Exit(1)
	}
	err = conf.Verify()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to verify config : %s\n", err.Error())
		os.Exit(1)
	}

	logger := initLoggerFromConfig(&conf.Logging)

	cfClient := cf.NewCfClient(&conf.Cf, logger.Session("cf"))
	err = cfClient.Login()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to login cloud foundry '%s'\n", conf.Cf.Api)
		os.Exit(1)
	}

	dopplerUrl := cfc.GetEndpoints().DopplerEndpoint

	logger.Info("create-noaa-client", map[string]interface{}{"dopplerUrl": dopplerUrl})
	handler.noaa = consumer.New(dopplerUrl, &tls.Config{InsecureSkipVerify: true}, nil)

	s := server.NewServer(conf.Server, cfClient, logger.Session("server"))
	s.Start()
}

func initLoggerFromConfig(conf *config.LoggingConfig) lager.Logger {
	logLevel, err := getLogLevel(conf.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to initialize logger: %s\n", err.Error())
		os.Exit(1)
	}
	logger := lager.NewLogger("as-metrics-collector")
	logger.RegisterSink(lager.NewWriterSink(os.Stdout, logLevel))

	return logger

}

func getLogLevel(level string) (lager.LogLevel, error) {
	switch level {
	case "debug":
		return lager.DEBUG, nil
	case "info":
		return lager.INFO, nil
	case "error":
		return lager.ERROR, nil
	case "fatal":
		return lager.FATAL, nil
	default:
		return -1, fmt.Errorf("Error: unsupported log level:%s", level)
	}
}

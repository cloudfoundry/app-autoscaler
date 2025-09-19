package startup

import (
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/configutil"
	"code.cloudfoundry.org/app-autoscaler/src/autoscaler/helpers"
	"code.cloudfoundry.org/lager/v3"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/sigmon"
)

type ConfigValidator interface {
	Validate() error
}

type ConfigWithLogging interface {
	ConfigValidator
	GetLogging() *helpers.LoggingConfig
}

type ConfigLoader[T ConfigWithLogging] func(path string, vcapConfigReader configutil.VCAPConfigurationReader) (T, error)

func ParseFlags() string {
	var path string
	flag.StringVar(&path, "c", "", "config file")
	flag.Parse()
	return path
}

func LoadVCAPConfiguration() (configutil.VCAPConfigurationReader, error) {
	vcapConfiguration, err := configutil.NewVCAPConfigurationReader()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to read vcap configuration : %s\n", err.Error())
	}
	return vcapConfiguration, err
}

func LoadAndValidateConfig[T ConfigWithLogging](path string, vcapConfig configutil.VCAPConfigurationReader, loader ConfigLoader[T]) (T, error) {
	var zero T
	conf, err := loader(path, vcapConfig)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to read config file '%s' : %s\n", path, err.Error())
		return zero, err
	}

	err = conf.Validate()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stdout, "failed to validate configuration : %s\n", err.Error())
		return zero, err
	}

	return conf, nil
}

func SetupEnvironment() {
	helpers.AssertFIPSMode()
	helpers.SetupOpenTelemetry()
}

func InitLogger(loggingConfig *helpers.LoggingConfig, serviceName string) lager.Logger {
	return helpers.InitLoggerFromConfig(loggingConfig, serviceName)
}

func StartServices(logger lager.Logger, members grouper.Members) error {
	monitor := ifrit.Invoke(sigmon.New(grouper.NewOrdered(os.Interrupt, members)))
	logger.Info("started")
	err := <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		return err
	}
	logger.Info("exited")
	return nil
}

func ExitOnError(err error, logger lager.Logger, message string, data ...lager.Data) {
	if err != nil {
		if len(data) > 0 {
			logger.Error(message, err, data[0])
		} else {
			logger.Error(message, err)
		}
		os.Exit(1)
	}
}

// Bootstrap provides a complete service initialization
func Bootstrap[T ConfigWithLogging](serviceName string, configLoader ConfigLoader[T]) (T, lager.Logger) {
	path := ParseFlags()
	vcapConfiguration, _ := LoadVCAPConfiguration()

	conf, err := LoadAndValidateConfig(path, vcapConfiguration, configLoader)
	if err != nil {
		os.Exit(1)
	}

	SetupEnvironment()
	logger := InitLogger(conf.GetLogging(), serviceName)

	return conf, logger
}

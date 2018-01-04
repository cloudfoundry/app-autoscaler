package commands

import (
	"cli/api"
	"cli/ui"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

type MetricsCommand struct {
	RequiredlArgs MetricsPositionalArgs `positional-args:"yes"`
	StartTime     string                `long:"start" description:"start time of metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to very beginning if not specified."`
	EndTime       string                `long:"end" description:"end time of the metrics collected with format \"yyyy-MM-ddTHH:mm:ss+/-HH:mm\" or \"yyyy-MM-ddTHH:mm:ssZ\", default to current time if not speficied."`
	Desc          bool                  `long:"desc" description:"display in descending order, default to ascending order if not specified."`
	Output        string                `long:"output" description:"dump the policy to a file in JSON format"`
}

type MetricsPositionalArgs struct {
	AppName    string `positional-arg-name:"APP_NAME" required:"true"`
	MetricName string `positional-arg-name:"METRIC_NAME" required:"true" description:"available metric name: \n memoryused, memoryutil, responsetime, throughput"`
}

func (command MetricsCommand) Execute([]string) error {

	switch command.RequiredlArgs.MetricName {
	case "memoryused":
	case "memoryutil":
	case "responsetime":
	case "throughput":
	default:
		return errors.New(fmt.Sprintf(ui.UnrecognizedMetricName, command.RequiredlArgs.MetricName))
	}

	var (
		st     int64 = 0
		et     int64 = time.Now().UnixNano()
		err    error
		writer *os.File
	)
	if command.StartTime != "" {
		st, err = parseTime(command.StartTime)
		if err != nil {
			return err
		}
	}
	if command.EndTime != "" {
		et, err = parseTime(command.EndTime)
		if err != nil {
			return err
		}
	}
	if st > et {
		return errors.New(fmt.Sprintf(ui.InvalidTimeRange, command.StartTime, command.EndTime))
	}

	if command.Output != "" {
		writer, err = os.OpenFile(command.Output, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		defer writer.Close()
	} else {
		writer = os.Stdout
	}

	return RetrieveMetrics(AutoScaler.CLIConnection,
		command.RequiredlArgs.AppName, command.RequiredlArgs.MetricName,
		st, et, command.Desc, writer)
}

func RetrieveMetrics(cliConnection api.Connection, appName, metricName string, startTime, endTime int64, desc bool, writer io.Writer) error {

	cfclient, err := api.NewCFClient(cliConnection)
	if err != nil {
		return err
	}
	err = cfclient.SetApp(appName)
	if err != nil {
		return err
	}

	endpoint, err := api.GetEndpoint()
	if err != nil {
		return err
	}
	if endpoint.URL == "" {
		return errors.New(ui.NoEndpoint)
	}

	apihelper := api.NewAPIHelper(endpoint, cfclient, os.Getenv("CF_TRACE"))

	ui.SayMessage(ui.ShowMetricsHint, appName)

	table := ui.NewTable(writer, []string{"Metrics", "Instance Index", "Value", "At"})
	var (
		page     uint64 = 1
		next     bool   = true
		noResult bool   = true
		data     [][]string
	)
	for next {
		next, data, err = apihelper.GetMetrics(metricName, startTime, endTime, desc, page)
		if err != nil {
			return err
		}

		for _, row := range data {
			table.Add(row)
		}
		if len(data) > 0 {
			noResult = false
			table.Print()
		}

		if !next {
			break
		}
		page += 1
	}

	if noResult {
		ui.SayOK()
		ui.SayMessage(ui.MetricsNotFound, appName)
	} else {
		if writer != os.Stdout {
			ui.SayOK()
		}
	}

	return nil
}

var timeFormats = []string{
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05-07:00",
}

func parseTime(input string) (ns int64, e error) {

	if input != "" {
		for _, format := range timeFormats {
			t, e := time.Parse(format, input)
			if e == nil {
				ns = t.UnixNano()
				return ns, nil
			}
		}
	}
	e = errors.New(fmt.Sprintf(ui.UnrecognizedTimeFormat, input))
	return
}

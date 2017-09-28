# Cloud Foundry CLI AutoScaler Plugin

AutoScaler plugin provides the command line interface to manage AutoScaler service policies, retrieve metrics and scaling history.

## Build and install

Run the following command to build the AutoScaler plugin
```
go build -o ascli cli
```

To install the plugin to cf cli
```
cf install-plugin ascli
```

To uninstall the plugin
```
cf uninstall-plugin AutoScaler
```

## Command List

| Command | Description |
|---------|-------------|
| [autoscaling-api, asa](#cf-autoscaling-api) | Set or view AutoScaler service API endpoint |
| [autoscaling-policy, asp](#cf-autoscaling-policy) | Retrieve the scaling policy of an application |
| [attach-autoscaling-policy, aasp](#cf-attach-autoscaling-policy) | Attach a scaling policy to an application |
| [detach-autoscaling-policy, dasp](#cf-detach-autoscaling-policy) | Detach the scaling policy from an application |
| [autoscaling-metrics, asm](#cf-autoscaling-metrics) | Retrieve the metrics of an application |
| [autoscaling-history, ash](#cf-autoscaling-history) | Retrieve the scaling history of an application|

## Command usage

### `cf autoscaling-api`

Set or view AutoScaler service API endpoint

```
cf autoscaling-api [URL] [--unset] 
```

**ALIAS**: asa

**OPTIONS**
- `--unset`: Unset the api endpoint

### `cf autoscaling-policy` 

Retrieve the scaling policy of an application

```
cf autoscaling-policy APP_NAME [--output PATH_TO_FILE]
```

**ALIAS**: asp


**OPTIONS**
- `--output` : dump the policy to a file in JSON format

### `cf attach-autoscaling-policy` 

Attach a scaling policy to an application
```
cf attach-autoscaling-policy APP_NAME PATH_TO_POLICY_FILE
```

**ALIAS**: aasp


### `cf detach-autoscaling-policy` 

Detach the scaling policy from an application
```
cf detach-as-policy APP_NAME
```
**ALIAS**: dasp


### `cf autoscaling-metrics`

Retrieve the metrics of an application

```
cf autoscaling-metrics APP_NAME METRIC_NAME [--start  START_TIME] [--end END_TIME] [--desc] [--output PATH_TO_FILE]
```
**ALIAS**: asm


**OPTIONS**
- `--start` : start time of metrics collected with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
- `--end` : end time of the metrics collected  with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
- `--desc` : display in descending order, default to ascending order if not specified
- `--output` : dump the metrics to a file in JSON format

###  `cf autoscaling-history` 

Retrieve the scaling history of an application.

```
cf autoscaling-history APP_NAME [--start  START_TIME] [--end END_TIME] [--desc] [--output PATH_TO_FILE]
```

**ALIAS**: ash

**OPTIONS**
- `--start` : start time of the scaling history with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
- `--end` : end time of the scaling history with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
- `--desc` : display in descending order, default to ascending order if not specified
- `--output` : dump the scaling history to a file in JSON format

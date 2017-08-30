# Cloud Foundry CLI AutoScaler Plugin

AutoScaler plugin provides the command line interface to manage AutoScaler service policies, retrieve metrics and scaling histories.

## Command List

| Command | Description |
|---------|-------------|
| [autoscaler-api, asa](#cf-autoscaler-api) | Set or view AutoScaler service API endpoint |
| [autoscaler-policy, asp](#cf-autoscaler-policy) | Retrieve the scaling policy of an application |
| [attach-autoscaler-policy, aasp](#cf-attach-autoscaler-policy) | Attach a scaling policy to an application |
| [detach-autoscaler-policy, dasp](#cf-detach-autoscaler-policy) | Detach the scaling policy from an application |
| [autoscaler-metrics, asm](#cf-autoscaler-metrics) | Retrieve the metrics of an application |
| [autoscaler-histories, ash](#cf-autoscaler-histories) | Retrieve the scaling histories of an application|

## Command usage

### `cf autoscaler-api`

Set or view AutoScaler service API endpoint

```
cf autoscaler-api [URL] [--unset] 
```

**ALIAS**: asa

**OPTIONS**
- `--unset`: Unset the api endpoint

### `cf autoscaler-policy` 

Retrieve the scaling policy of an application

```
cf autoscaler-policy APP_NAME [-f PATH_TO_FILE]
```

**ALIAS**: asp


**OPTIONS**
- `-f` : dump the policy to a file in JSON format

### `cf attach-autoscaler-policy` 

Attach a scaling policy to an application
```
cf attach-autoscaler-policy APP_NAME PATH_TO_POLICY_FILE
```

**ALIAS**: aasp


### `cf detach-autoscaler-policy` 

Detach the scaling policy from an application
```
cf detach-as-policy APP_NAME
```
**ALIAS**: dasp


### `cf autoscaler-metrics`

Retrieve the metrics of an application

```
cf autoscaler-metrics APP_NAME METRIC_NAME [--start  START_TIME] [--end END_TIME] [--desc] [-f PATH_TO_FILE]
```
**ALIAS**: asm


**OPTIONS**
- `--start` : start time of metrics collected with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
- `--end` : end time of the metrics collected  with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
- `--desc` : display in descending order, default to ascending order if not specified
- `-f` : dump the metrics to a file in JSON format

###  `cf autoscaler-histories` 

Retrieve the scaling histories of an application.

```
cf autoscaler-histories APP_NAME [--start  START_TIME] [--end END_TIME] [--desc] [-f PATH_TO_FILE]
```

**ALIAS**: ash

**OPTIONS**
- `--start` : start time of the scaling histories with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to very beginning if not specified.
- `--end` : end time of the scaling histories with format "yyyy-MM-ddTHH:mm:ss+/-HHmm" or "yyyy-MM-ddTHH:mm:ssZ", default to current time if not speficied.
- `--desc` : display in descending order, default to ascending order if not specified
- `-f` : dump the scaling histories to a file in JSON format

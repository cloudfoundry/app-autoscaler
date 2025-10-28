# ASATS test app

This app is used in the app-autoscaler acceptance test suite (ASATS) to generate metrics to test end-to-end scenarios.

The app can be made to consume and release on demand:
- main memory 
- CPU
- hard disk

It can be used to emit custom metrics.

It can be used to respond to HTTP requests:
- as fast as possible
- or with a configurable request latency

**The [API](https://github.com/cloudfoundry/app-autoscaler-release/blob/main/src/acceptance/assets/app/go_app/internal/app/app.go#L49-L54) of the test app is not part of the public API and may change at any time.**

Still, the app can be useful during development and one such use case is described in detail in the next section.

## Deploy Test Application with Custom Metrics

### Sample Autoscaling Policy

Autoscaling policy is available at [test-policy.json](example/test-policy.json)

```json
{
  "instance_min_count": 1,
  "instance_max_count": 2,
  "scaling_rules": [
    {
      "metric_type": "review_count",
      "breach_duration_secs": 60,
      "threshold": 200,
      "operator": ">=",
      "cool_down_secs": 60,
      "adjustment": "+1"
    },
    {
      "metric_type": "review_count",
      "breach_duration_secs": 60,
      "threshold": 100,
      "operator": "<=",
      "cool_down_secs": 60,
      "adjustment": "-1"
    }
  ]
}
```

### Deploy Test Application

Prepare acceptance_config.json. Example [acceptance_config.json](../../../example_config/acceptance_config.json)

```bash
CONFIG=$PWD/acceptance_config.json make deploy-test-app && \
#attach autoscaling policy (as defined above)
cf attach-autoscaling-policy test_app test-policy.json && \
# show autoscaling policy using autoscaler cli plugin
cf asp test_app
```

### Send Custom Metrics

```bash
# scale out 
APP_ROUTE=$(cf app test_app | grep routes | awk '{print $2}')
curl https://$APP_ROUTE/custom-metrics/mtls/review_count/201

# scale In 
APP_ROUTE=$(cf app test_app | grep routes | awk '{print $2}')
curl https://$APP_ROUTE/custom-metrics/mtls/review_count/78
```


### View Metrics in Log Cache

Metrics are ingested into log-cache and can be viewed using [CF Log Cache CLI Plugin](https://github.com/cloudfoundry/log-cache-cli)

```bash
cf tail --name-filter="review_count" test_app --follow
```

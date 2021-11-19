
# App AutoScaler User Guide

This document describes how to configure and use [Cloud Foundry App Auto-Scaler][git] to automatically scale your application **horizontally**  based on the scaling policy that you define.

---
## Overview

The [Cloud Foundry App Auto-Scaler][git] automatically adjust the instance number of Cloud Foundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

The Cloud Foundry [Admin or Space Developers role][userrole] is needed to manage the autoscaling policy, query metric values and scaling events. 

---
## Concepts of Autoscaling policy

Autoscaling policy is represented in JSON and consists of the following parts. Refer to the [policy specification][policy] for the detailed definition.

Autoscaling can be defined explicitly in the service binding or through a [default policy](#default-policy) at a service instance level.

### Default policy 

The default policy feature is optional. It applies to all service bindings 
that are created without an explicit policy. It can be configured when provisioning a
service instance. It can be specified by passing through a service
creation parameter called `default_policy`.

The default policy can be changed via the `cf update-service` command. The changes
will be propagated to all apps that use the default policy. If a policy is set explicitly 
on an app with the same content as the default policy, updates on the default policy 
will not affect that app. Setting the default policy can be achieved by removing 
the binding level policy.

The default policy can be removed, by setting it to an empty JSON object when 
calling `cf update-service`.



### Instance limits

Instance limits are used to define the default minimal and maximal instance number of your application.

### Dynamic scaling rules

####  Metric types

The following are the built-in metrics that you can use to scale your application based on. They are averaged over all the instances of your application. 

* **memoryused**
	
	"memoryused" represents the absolute value of the used memory of your application. The unit of "memoryused" metric is "MB".
	
* **memoryutil**
	
	"memoryutil", a short name of "memory utilization", is the used memory of the total memory allocated to the application in percentage.
	
	For example, if the memory usage of the application is 100MB  and memory quota is 200MB, the value of "memoryutil" is 50%.

* **cpu**

	"cpu", a short name of "cpu utilization", is the cpu usage of your application in percentage.

* **responsetime**
	
	"responsetime" represents the average amount of time the application takes to respond to a request in a given time period.  The unit of "responsetime" is "ms" (milliseconds).

* **throughput**

	"throughput" is the total number of the processed requests  in a given time period. The  unit of "throughput" is "rps" (requests per second).

* **custom metric** 

	Custom metric is supported since [app-autoscaler v3.0.0 release][app-autoscaler-v3.0.0]. You can define your own metric name and emit your own metric to `App Autoscaler` to trigger further dynamic scaling. Only alphabet letters, numbers and "_" are allowed for a valid metric name, and the maximum length of the metric name is limited up to 100 characters. 
	
 
####  Threshold and Adjustment

`App AutoScaler` evaluates the aggregated metric values against the threshold defined in the dynamic scaling rules, and change the application instance count according to the adjustment setting.  

For example, if you want to scale out your application by adding 2 instances when the application's througput exceeds 800 requests per second, define the the scaling rule as below:
```
{
  "metric_type": "throughput",
  "operator": ">=",
  "threshold": 800,
  "adjustment": "+2"
}
```

####  (Optional) Breach duration and Cooldown 

`App AutoScaler` will not take scaling action until your application continues breaching the rule in a time duration defined in `breach_duration_secs`.  This setting controls how fast the autoscaling action could be triggered. 

`cool_down_secs` defines the time duration to wait before the next scaling kicks in.  It helps to ensure that your application does not launch or terminate instances before your application becomes stable. This setting can be configured based on your instance warm-up time or other needs.

*Note:*

* You can define multiple scaling-out and scaling-in rules. However, `App-AutoScaler` does not detect conflicts among them.  It is your responsibility to ensure the scaling rules do not conflict with each other to avoid fluctuation or other issues. 

* `breach_duration_secs` and `cool_down_secs` are both optional entries in scaling_rule definition.  The `App Autoscaler` provider will define the default value if you omit them from the policy.

###  Schedules

`App AutoScaler` uses schedules to overwrite the default instance limits for specific time periods. During these time periods, all dynamic scaling rules are still effective. 

The design of `Schedules` is mainly used to prepare enough instances for  peak hours. 

You can define recurring schedules, or specific schedules which are executed only once in [autoscaling policy][policy].

Particularly, besides overriding the default instance limit of `instance_min_count` and `instance_max_count`, you can also define an `initial_min_instance_count` in the schedule.  

For example, in the following schedule rule, `App AutoScaler` will set your application instance number to be 50 when the schedule starts, and make sure your instances number is within [10,100] during the scheduled time period.  
```
{
    "start_date_time": "2099-01-01T10:00",
    "end_date_time": "2099-01-01T20:00",
    "instance_min_count": 10,
    "instance_max_count": 100,
    "initial_min_instance_count": 50
}
```

----
## Create Autoscaling Policy JSON File

The following gives some policy examples for you to start with. Refer to [Policy speficication][policy] for the detailed JSON format of the autoscaling policy.

* [Autoscaling policy example for dynamic scaling rules][policy-dynamic]
* [Autoscaling policy example for custom metrics ][policy-dynamic-custom]
* [Autoscaling policy example for both dynamic scaling rules and schedules][policy-all]

## Create Autoscaling Policy JSON File
----

## Connect an application to App-AutoScaler

`App-AutoScaler` can be offered as a Cloud Foundry service or an extension of your Cloud Foundry platform. Consult your Cloud Foundry provider for how it is offered. 

###  As a Cloud Foundry extension
When `App AutoScaler` is offered as Cloud Foundry platform extension,  you don't need to connect your application to autoscaler, go directly to next section to attach autoscaling policy to your application with CLI. 

###  As a Cloud Foundry service
When `App AutoScaler` is offered as a Cloud Foundry service via [open service broker api][osb] , you need to provision and bind `App AutoScaler` service through [Cloud Foundry CLI][cfcli]  first. 

* [Create an instance of the service][sprovision]
* [Bind the service instance to your application][sbind]
	
	Note you can attach scaling policy together with service binding by providing the policy file name as a parameter of the service binding command. 
	```
	cf bind-service <app_name> <service_instance_name> -c <policy_file_name>
	```	

To disconnect `App AutoScaler` from your application, unbind the service instance. This will remove the  autoscaling policy as well. Furthermore, you can deprovision the service instance if no application is bound.

* [Unbind the service instance from your application][sunbind]
* [Delete service instance][sdeprovision]


----
## Command Line interface

This section gives how to use the command line interface to manage autoscaling policies,  query metrics and scaling event history. Go to the [CLI user guide][cli] for the detailed CLI references. 

### Getting started with AutoScaler CLI 

* Install [AutoScaler CLI plugin][cli]
* Set App AutoScaler API endpoint （Optional)
    
    AutoScaler CLI plugin interacts with `App AutoScaler`  through its [public API][api].  
	
	By default, `App AutoScaler` API endpoint is set to `https://autoscaler.<cf-domain>` automatically.  You can change to others like the example below
    ``` 
	 cf asa https://example-autoscaler.<cf-domain>
    ```

### Attach policy 

Create or update autoscaling policy for your application with command. 
```
	 cf aasp <app_name> <policy_file_name>
```

### Detach policy 

Remove autoscaling policy to disable `App Autoscaler` with command
```
	 cf dasp <app_name>
```

### View policy 

To retrieve the current autoscaling policy, use command below
```
	 cf asp <app_name>
```

### Query metrics

Query the most recent metrics with command
```
	 cf asm <app_name> <metric_type>
```

Note the output of the `cf asm` command shows aggregated metrics instead of the raw data of instance metrics.  

Refer to  [AutoScaler CLI user guide][cli] for the advanced options to specify the time range, the number of metric values to return and display order.


### Query scaling events

To query your application's scaling events, use command below
```
	 cf ash <app_name>
```

Refer to  [AutoScaler CLI user guide][cli] for advanced options to specify the time range, the number of events to return and display order.

### Create autoscaling credential before submitting custom metric

Create custom metric credential for an application. The credential will be displayed in JSON format.
```
cf create-autoscaling-credential <app_name>
```
Refer to  [AutoScaler CLI user guide][cli] for more details.

### Delete autoscaling credential
Delete custom metric credential when unncessary.
```
cf delete-autoscaling-credential <app_name>
```

----
## Auto-scale your application with custom metrics

With custom metric support,  you can scale your application with your own metrics with below steps.

* Claim custom metric in your policy

First, you need to define a dynamic scaling rule with a customized metric name, refer to [Autoscaling policy example for custom metrics][policy-dynamic-custom].

*  Create credential for your application

To scale with custom metric, your application need to emit its own metric to `App Autoscaler`'s metric server.  
Given the metric submission is proceeded inside an application,  an `App Autoscaler` specific credential is required to authorize the access.

If `App Autoscaler` is offered as a service,  the credential and autoscaler metric server's URL are injected into VCAP_SERVICES by service binding directly.

If `App Autoscaler` is offered as a Cloud Foundry extension, the credential need to be generated explictly with command  `cf create-autoscaling-credential` as below example: 

```
>>>cf create-autoscaling-credential <app_name> --output <credential_json_file_path> 
...
>>> cat <credential_json_file_path> 
{
    "app_id": "c99f4f6d-2d67-4eb6-897f-21be90e0dee5",
    "username": "9bb48dd3-9246-4d7e-7827-b478e9bbedcd",
    "password": "c1e47d80-e9a0-446a-782b-63fe9f974d4c",
    "url": "https://autoscalermetrics.bosh-lite.com"
}
```

Then, you need to configure the credential to your application as an environment variable by `cf set-env` command 
or through user-provided-service approach as below: 

```
>>> cf create-user-provided-service <user-provided-service-name> -p <credential_json_file_path> 
...
>>> cf bind-service <app_name>  <user-provided-service-name> 
...
TIP: Use 'cf restage <app-name>' to ensure your env variable changes take effect
```
With the user-provided-service aproach, you can consume the credential from VCAP_SERVICES environments.

* Emit your own metrics to autoscaler

You need to emit  your own metric for scaling to the "URL" specified in credential JSON file with below API endpoint.

```
POST /v1/apps/:guid/metrics
```

*Note:* `:guid` is the `app_id` of your application.  You can get it from the credential JSON file , or from the environment variable

A JSON payload is required with above API to submit metric name, value and the correspondng instance index.
```
  {
    "instance_index": <INSTANCE INDEX>,
    "metrics": [
      {
        "name": "<CUSTOM METRIC NAME>",
        "value": <CUSTOM METRIC VALUE>,
        "unit": "<CUSTOM METRIC UNIT>",
      }
    ]
  }
```

*Note:*

* `<INSTANCE INDEX>` is the index of current application instance. You can fetch the index from environment variable `CF_INSTANCE_INDEX`
* `<CUSTOM METRIC NAME>` is the name of the emit metric which must be equal to the metric name that you define in the policy. 
* `<CUSTOM METRIC VALUE>` is value that you would like to submit. The `value` here must be a NUMBER.
* `<CUSTOM METRIC UNIT>` is the unit of the metric, optional.

Please refer to [Emit metric API Spec][emit-metric-api] for more information.


[git]:https://github.com/cloudfoundry/app-autoscaler
[cli]: https://github.com/cloudfoundry/app-autoscaler-cli-plugin#install-plugin
[policy]: policy.md
[policy-dynamic]: dynamicpolicy.json
[policy-dynamic-custom]: customemetricpolicy.json
[policy-all]: fullpolicy.json
[api]: Public_API.rst
[osb]: https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md
[cfcli]: https://github.com/cloudfoundry/cli
[sprovision]: https://docs.cloudfoundry.org/devguide/services/managing-services.html#create
[sbind]: https://docs.cloudfoundry.org/devguide/services/managing-services.html#bind
[sunbind]:https://docs.cloudfoundry.org/devguide/services/managing-services.html#unbind
[sdeprovision]:https://docs.cloudfoundry.org/devguide/services/managing-services.html#delete
[userrole]:https://docs.cloudfoundry.org/concepts/roles.html#spaces
[app-autoscaler-v3.0.0]: https://bosh.io/releases/github.com/cloudfoundry-incubator/app-autoscaler-release?all=1#latest
[emit-metric-api]:https://github.com/cloudfoundry/app-autoscaler/blob/develop/docs/Public_API.rst#submit-custom-metric-to-autoscaler-metric-server

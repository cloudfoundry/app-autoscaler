
# App AutoScaler User Guide

This document describes how to configure and use [Cloud Foundry App Auto-Scaler][git] to automatically scale your application **horizontally**  based on the scaling policy that you define.

---
## Overview

The [Cloud Foundry App Auto-Scaler][git] automatically adjust the instance number of Cloud Foundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

The Cloud Foundry [Admin or Space Developers role][userrole] is needed to manage the autoscaling policy, query metric values and scaling events. 

---
## Autoscaling policy

Autoscaling policy is represented in JSON and consists of the following parts. Refer to the [policy specification][policy] for the detailed definition.

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

####  Breach duration and Cooldown

`App AutoScaler` will not take scaling action until your application continues breaching the rule in a time duration defined in `breach_duration_secs`.  This setting controls how fast the autoscaling action could be triggered. 

`cool_down_secs` defines the time duration to wait before the next scaling kicks in.  It helps to ensure that your application does not launch or terminate instances before your application becomes stable. This setting can be configured based on your instance warm-up time or other needs.

*Note:*

You can define multiple scaling-out and scaling-in rules. However, `App-AutoScaler` does not detect conflicts among them.  It is your responsibility to ensure the scaling rules do not conflict with each other to avoid fluctuation or other issues. 


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
## Create Autoscaling policy

The following gives some policy examples for you to start with. Refer to [Policy speficication][policy] for the detailed JSON format of the autoscaling policy.

* [Autoscaling policy with dynamic scaling rules][policy-dynamic]
* [Autoscaling policy with dynamic scaling rules and schedules][policy-all]

----

## Connect an application to App-AutoScaler

`App-AutoScaler` can be offered as a Cloud Foundry service or an extension of your Cloud Foundry platform. Consult your Cloud Foundry provider for how it is offered. 

###  As a Cloud Foundry extension
When `App AutoScaler` is offered as Cloud Foudnry platform extension,  you don't need to connect your application to autoscaler, go directly to next section on how to configure your policy.

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
* Set App AutoScaler API endpoint （optional)
    
    AutoScaler CLI plugin interacts with `App AutoScaler`  through its [public API][api].  
	
	By default, `App AutoScaler` API endpoint is set to `https://autoscaler.<cf-domain>` automatically.  You can change to others like the example below
    ``` 
	 cf asa https://example-autoscaler.<cf-domain>
    ```

### Attach policy 

Create or update auto-scaling policy for your application with command
```
	 cf aasp <app_name> <policy_file_name>
```

### Detach policy 

Remove auto-scaling policy to disable `App Autoscaler` with command
```
	 cf dasp <app_name>
```

### View policy 

To retrieve the current auto-scaling policy, use command below
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


[git]:https://github.com/cloudfoundry-incubator/app-autoscaler
[cli]: https://github.com/cloudfoundry-incubator/app-autoscaler-cli-plugin#install-plugin
[policy]: policy.md
[policy-dynamic]: dynamicpolicy.json
[policy-all]: fullpolicy.json
[api]: Public_API.rst
[osb]: https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md
[cfcli]: https://github.com/cloudfoundry/cli
[sprovision]: https://docs.cloudfoundry.org/devguide/services/managing-services.html#create
[sbind]: https://docs.cloudfoundry.org/devguide/services/managing-services.html#bind
[sunbind]:https://docs.cloudfoundry.org/devguide/services/managing-services.html#unbind
[sdeprovision]:https://docs.cloudfoundry.org/devguide/services/managing-services.html#delete
[userrole]:https://docs.cloudfoundry.org/concepts/roles.html#spaces


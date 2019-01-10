
# Scaling an Application Using App AutoScaler
This topic describes how to configure and use [Cloud Foundry App Auto-Scaler][git] to scale your application **horizontally** and automatically based on the scaling policy that you set.

---
## Overview

The [Cloud Foundry App Auto-Scaler][git] provides the capacity to : 
* Adjust the application's instance number dynamically based on application metrics' thresholds, such as memory usage, throughput, response time and etc.
* Specific the default application instance limit  or configure various sets of instance limit with different schedules based on time. 

The Cloud Foundry [Admin and Space Developers][userrole] are authorized to use `App AutoScaler` to configure the autoscaling behavior of an application by attaching an [autoscaling policy][policy].

---
## Basic Concepts

`App AutoScaler` performs each scaling action based on the [autoscaling policy][policy] attached to your application.  

[Autoscaling policy][policy] is constructed with the following parts: 

### Instance limit

Instance limit is used to control the range in which auto-scaling would happen. 
You need to define proper values for `instance_min_count` and `instance_min_count` in [autoscaling policy][policy].

### Dynamic scaling rules

The dyanmic scaling rules are used to define how the application would be scaled in response to the changing workload.

####  Supported metric types
Dynamic scaling would happen based on different application metric types, including:  

* **memoryused**
	
	The metric "memoryused" represents the **average** used memory value of your application for all its instances. 
	
	The default unit of "memoryused" metric is "MB".
	
* **memoryutil**
	
	The metric "memoryutil", a short name of "memory utilization", is the **average** percentage of used memory of your application for all its instances. 
	
	For example, if the memory usage of the application is "100MB of 200MB", the value of "memoryutil" is 50%.

* **responsetime**
	
	The metric "responsetime" represents the **average** latency for the processed requests occurred in a period time of your application for all its instances. 
	
	The default unit of "responsetime" is "ms" (milliseconds).

* **throughput**

	The metric "throughput" is the total number of the processed requests occurred in in a period time of your application for all its instances. 
	
	The default unit of "throughput" is "rps" (request per second).

*Note:*

You can define multiple scale-out and scale-in rules. However, `App AutoScaler` responses to any breached rule regardless any possible conflict within them.  It is your responsibility to ensure the scaling rules do not conflict with each other to avoid fluctuation or other issues. 
 
 
####  Threshold and Adjustment

After the application instances' metrics are collected, `App AutoScaler` aggregates and evaluates the metrics against the thresholds you set , then change the application instance count according to the adjustment setting.  

[Autoscaling policy][policy] allows you to customize threshold, adjustment and etc.

For example, if you want to trigger a scaling-out action when the application's actual througput exceeds the defined threshold,  you can set its `threshold` and desired `adjustment` with the following policy definition:
```
{
  "metric_type": "throughput",
  "operator": ">=",
  "threshold": 800,
  "adjustment": "+2"
}
```

####  Breach duration and Cooldown

`App AutoScaler` also provides `breach_duration_secs` and  `cool_down_secs` options to customize scaling behaviors. 

The concept "breach duration" means that the `App AutoScaler` won't take the desired adjustment until the breach lasts for a period longer than `breach_duration_secs` setting.  This parameter controls how fast the autoscaling action could be triggered. 

The concept "cooldown" defines a pause between different scaling actions. With a proper `cool_down_secs` setting, your application can have a chance to rebalance its workloads and keep stable before next scaling action is triggered. 


###  Schedules

`App AutoScaler` uses schedules to adjust your application's instance limit during a pre-defined period while all dynamic scaling rules are 
still effective. 

The design of `Schedules` aims to scale your application with a larger range in the particular period to handle a higher workload.  It is mainly used when application resource demand is predictable. 

You can define repeated recurring schedules and a one-time specific date schedules in [autoscaling policy][policy].

In particular,  besides overriding the default instance limit of `instance_min_count` and `instance_max_count` of an application you can also define an `initial_min_instance_count` in schedule definition.  

For example, in the following schedule definition, `App AutoScaler` will ensure your application instance number to be equal or greater than the  `initial_min_instance_count` limit when the schedule is kicked off.  
```
{
    "start_date_time": "2099-01-01T10:00",
    "end_date_time": "2099-01-01T20:00",
    "instance_min_count": 10,
    "instance_max_count": 100,
    "initial_min_instance_count": 50
}
```
Then, during the scheduled window, your application instances maybe vary between `instance_min_count` and `instance_max_count` according to its actual workload. 

----
## Build Autoscaling policy from scratch

[Autoscaling policy][policy] is written in JSON format. Please refer to [Policy speficication][policy] to understand the details.

To get started with `App AutoScaler` more quickly, you can build own policies based on the following examples: 

* [Autoscaling policy with dynamic scaling rules][policy-dynamic]
* [Autoscaling policy with dynamic scaling rules and schedules][policy-all]

----

## Connect an application to App-AutoScaler

`App-AutoScaler` could be offered as a Cloud Foundry service or a build-in capability per different service provider.  Please consult your Cloud Foundry provider to understand how it is offered, and then setup `App AutoScaler` to your application accordingly. 

###  As build-in application capability
When `App AutoScaler` is offered as a build-in application capability,  you don't need to do anything in this step.  Please go ahead to next section directly to configure your policy.

###  As App-AutoScaler service
When `App AutoScaler` is offered as a Cloud Foundry service via [open service broker api][osb] , you need to provision and bind `App AutoScaler` service through [Cloud Foundry CLI][cfcli]  first. 

* [Create an instance of the service][sprovision]
* [Bind the service instance to your application][sbind]
	
	Furthermore, you can attach the scaling policy JSON file along with service binding operation by providing the JSON file name as a parameter of service binding command. 
	```
	cf bind-service <app_name> <service_instance_name> -c <policy>
	```	

To remove `App AutoScaler` from your application, you can unbind the service instance which will remove the existing autoscaling policy as well. Furthermore, you can deprovision the service instance.
* [Unbind the service instance from your application][sunbind]
* [Delete service instance][sdeprovision]


----
## Attach Autoscaling Policy

You can always use the [App Autoscaler command-line interface (CLI) plugin][cli] (aka `AutoScaler CLI`) to configure policy,  query metrics and histories from `App AutoScaler`. 

This section is a briefing for the usage of `AutoScaler CLI` .  For more options and complete usage, please refer to the [AutoScaler CLI user guide][cli].

### Getting started with AutoScaler CLI 

* Install [AutoScaler CLI][cli]
* Set App AutoScaler API endpoint （optional)
    
    [AutoScaler CLI ][cli] interact  with `App AutoScaler`  through its public API.  
	
	By default, `App AutoScaler` API endpoint is set to `autoscaler.<domain>` automatically.  You can change it with command: 
```
	 cf asa https://autoscaler.<domain>
```

### Attach policy 
You can create or update auto-scaling policy for your application with command:
```
	 cf aasp <your app> <your policy>
```

### Detach policy 
You can remove auto-scaling policy to disable `App Autoscaler` with command:
```
	 cf dasp <your app>
```

### View existing policy 
To view your existing auto-scaling policy,  please use command:
```
	 cf asp <your app>
```

### View application metrics
You can query the most recent application metrics with command
```
	 cf asm <your app> <metric type>
```
The output of the `cf asm` command is designed to be application's aggregated metrics given `App AutoScaler` makes decisions based on the  aggregated average value rather than individual raw data of each instance metric.  

With advanced options documented in the [AutoScaler CLI user guide][cli], you can define the query range and display approach.

### View application scaling histories
For audit purpose,  you can query your application's scaling history  with command
```
	 cf ash <your app>
```
 

[git]:https://github.com/cloudfoundry-incubator/app-autoscaler
[cli]: https://github.com/cloudfoundry-incubator/app-autoscaler-cli-plugin/blob/master/README.md
[policy]: https://github.com/cloudfoundry-incubator/blob/master/docs/policy.md
[policy-dynamic]: https://github.com/cloudfoundry-incubator/blob/master/docs/dynamicpolicy.json
[policy-all]: https://github.com/cloudfoundry-incubator/blob/master/docs/fullpolicy.json
[osb]: https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md
[cfcli]: https://github.com/cloudfoundry/cli
[sprovision]: https://docs.cloudfoundry.org/devguide/services/managing-services.html#create
[sbind]: https://docs.cloudfoundry.org/devguide/services/managing-services.html#bind
[sunbind]:https://docs.cloudfoundry.org/devguide/services/managing-services.html#unbind
[sdeprovision]:https://docs.cloudfoundry.org/devguide/services/managing-services.html#delete
[userrole]:https://docs.cloudfoundry.org/concepts/roles.html#spaces


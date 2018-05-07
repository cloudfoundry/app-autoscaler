# App-AutoScaler End-User Manual

The `App-AutoScaler` provides the capability to adjust the computation resources for Cloud Foundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

----
## Terminology

### Scaling type
* Dynamic scaling

    With dynamic scaling, the runtime performance of your application will be  monitored, and evaluated to see whether a scling in/out action is required, therefore the scaling decision is made by `App-AutoScaler` on demand. 

* Schedued scaling

    Scheduled based scaling is designed to scale in/out your application instances at a predefined time slot. It is useful when the workload of your application is predictable.  

### Support metric type

Four metric types are supported now.

* memoryused 

    The metric "memoryused" is the absolute value of used memory of an application.
    The default unit of "memoryused" is "MB". 

* memoryutil

    The metric "memoryutil", a short name of "memory utilization", is the relative value of used memory and the total memory of an application.
    If the memory usage of the application is "100MB of 200MB", the value of "memoryutil" is 50%.  
    The default unit of "memoryutil" is "%".

* responsetime

    The metric "responsetime" is an average value of the aggregated total elapsed time of all processed requests occurred in a specific window (aka, collection interval). 
    The default unit of "responsetime" is "ms" (milliseconds).

* throughput 

    The metric "througput" is the total number of the processed requests occurred in a specific window (aka, collection interval). 
    The default unit of "throughput" is "rps" (request per second).

Note: 
You can define multiple scaling rules for more than one metric type. However, the `App-AutoScaler` does not detect conflicts between scaling policies. When you define the scaling policy, you must ensure that multiple scaling rules do not conflict with one another. Otherwise, you might see the total instance number fluctuates because the application is scaled in at first and scaled out then.

### Policy definition

Scaling policy is essential to play with `App-AutoScaler` properly. 

Refer to [Policy Definition][aa] for details and sample policy file. 

----
## Manage App-AutoScaler service

`App-AutoScaler` is offered as a Cloud Foundry service via [open service broker api][ab]. You can provision and bind service through [Cloud Foundry CLI][ac]. 

### Provision & bind service 
As a Space Developer, you can create the service instance, and then bind your application to the service instance.
```
cf create-service autoscaler  autoscaler-free-plan  <service_instance_name>
cf bind-service <app_name> <service_instance_name>
```

Alternatively, if you have a policy ready for use before service binding, you can attach the policy as a parameter of service binding.
```
cf bind-service <app_name> <service_instance_name> -c <policy>
```

### Manage service with public API

Check [public api definition][af] for details. 

### Manage service with command line tool 

Go to [app-autoscaler-cli-plugin][ad] project to manage `App-AutoScaler` service with CLI tool.
Download available at [CF Plugin Community][ae] as well.

With [app-autoscaler-cli-plugin][ad] installed, you can manage policy, retrive metrics and scaling history from CLI easily.


----
## Things you need to know before auto-scale an application

Before using `App-AutoScaler`, a series of performance engineering work is strongly recommended, so that you can understand the performance of your application and have a good predict of the benefit when enable the service.

You might consider to take the following steps to create a proper scaling policy: 

* Understand the traffic and workload type of the application
* Benchmark the application to understand the performance of the application
* Scale application manually to understand how the application behaves when scaling out ( how long it is needed to warm up, what is the impact of existing load and sticky session if it is used)
* Identify the performance bottleneck from the benchmarking result, and decide which metric should be used to dynamically adjust the instance number.
* Define the initial scaling policy and enable the service
* Simulate the workload and test with the service to adjust detailed settings of the policy, including thresholds, steps of scaling, statistics window, breach duration and cool-down period
* Define scheduled scaling for peak hours 
* Simulate the peak hour workload, and adjust the min/max instance number settings in the scheduled policy
* Apply the refined policy and let it go

Also, here are some general guidelines:

* Do not use `App-AutoScaler` service to handle sudden burst. If the burst is predictable,use scheduled scaling to get enough resource prepared
* Prevent excessive scale in/out by setting min/max instance number 
* Allow enough quota of your organization for scaling out
* Carefully set threshold, don’t push too high,not to exceed memory limit as much as possible 
* Scale down less aggressively
* For CPU intensive app, be aware of that CPU is actually weighted shared with other apps on the same host vm.  Use other metrics like 
throughput/response time as metrics
* Design dynamic scaling rules carefully when using multiple metric types


[aa]: https://github.com/cloudfoundry-incubator/app-autoscaler/blob/develop/docs/Policy_definition.rst
[ab]: https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md
[ac]: https://github.com/cloudfoundry/cli
[ad]: https://github.com/cloudfoundry-incubator/app-autoscaler-cli-plugin
[ae]: https://plugins.cloudfoundry.org/
[af]: https://github.com/cloudfoundry-incubator/app-autoscaler/blob/develop/docs/Public_API.rst

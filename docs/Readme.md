# App-AutoScaler User Guide

The `App-AutoScaler` provides the capability to automatically adjust the instance number of Cloud Foundry applications through

* Dynamic scaling based on application performance metrics
* Scheduled scaling based on time

----
## Terminology

### Scaling type
* Dynamic scaling

    Dynamic scaling adjusts the application instance number based on the rules defined on performance metrics. It is used to scale application in response to dynamically changing workload

* Scheduled scaling

    Scheduled scaling adjusts the application instance number at a predefined time slot. It is mainly used when application resource demand is predictable.  

### Metrics supported for dynamic scaling

The following metrics are supported right now. More metrics and custom metrics will be supported in the near future. 

* memoryused 

    The metric "memoryused" is the absolute value of used memory of an application instance.
    
    The unit of "memoryused" is "MB". 

* memoryutil

    The metric "memoryutil", a short name of "memory utilization", is the percentage of used memory for total memory allocated to an application instance. If the memory usage of the application is "100MB of 200MB", the value of "memoryutil" is 50%.  


* responsetime

    The metric "responsetime" is an average value of the aggregated total elapsed time of all processed requests occurred in a specific time window (aka, collection interval) for an application instance.

    The unit of "responsetime" is "ms" (milliseconds).

* throughput 

    The metric "througput" is the total number of the processed requests occurred in a specific time window (aka, collection interval) for an application instance.
    
    The unit of "throughput" is "rps" (request per second).

Note: 

You can define multiple scaling-out and scaling-in rules. However, `App-AutoScaler` does not detect conflicts among them. When you define the scaling policy, you need to ensure the scaling rules do not conflict with each other. Otherwise, you might experience the fluctuations of the application instances.

### Policy definition

Scaling policy is essential to play with `App-AutoScaler` properly. 

Refer to [Policy Definition][aa] for details and sample policy file. 

----
## Use `App-AutoScaler` service

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

### Use public API

Check [public api definition][af] for details. 

### Use command line tool 

Go to [app-autoscaler-cli-plugin][ad] project to manage `App-AutoScaler` service with CLI tool.
Download available at [CF Plugin Community][ae] as well.

With [app-autoscaler-cli-plugin][ad] installed, you can manage policy, retrive metrics and scaling history from CLI easily.


----
## Things you need to know before auto-scale an application

Before using `App-AutoScaler`, a series of performance engineering work is strongly recommended, so that you can understand the workload characteristics of your application and set the proper scaling policy.

You might consider to take the following steps when creating the scaling policy: 

* Benchmark the application to understand the performance
* Identify the performance bottleneck from the benchmarking result, and decide which metric should be used to dynamically adjust the instance number.
* Scale application manually to understand how the application behaves when scaling out/in 
* Define the initial scaling policy and enable the service
* Drive load and test with the policy to see how it works. Adjust detailed settings of the policy, including thresholds, steps of scaling, breach duration and cool-down period
* Define scheduled scaling for peak hours 
* Simulate the peak hour workload, and adjust the min/max instance number settings in the scheduled policy
* Apply the refined policy 

Also, here are some general guidelines:

* Do not use `App-AutoScaler` service to handle sudden burst. If the burst is predictable, use scheduled scaling to prepare enough application instances for it.
* Prevent excessive scale in/out by setting min/max instance number 
* Allow enough quota of your organization for scaling out
* Carefully set the threshold, don’t push too high. Make sure it will not exceed memory limit 
* Scale down less aggressively
* For CPU intensive app, be aware of that CPU is actually weighted shared with other apps on the same host vm. Use other metrics like throughput/responsetime for scaling
* Design dynamic scaling rules carefully when using multiple metrics


[aa]: Policy_definition.rst
[ab]: https://github.com/openservicebrokerapi/servicebroker/blob/master/spec.md
[ac]: https://github.com/cloudfoundry/cli
[ad]: https://github.com/cloudfoundry-incubator/app-autoscaler-cli-plugin
[ae]: https://plugins.cloudfoundry.org/
[af]: Public_API.rst

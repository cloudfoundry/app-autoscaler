# App AutoScaler Policy JSON Specification 

`App AutoScaler` requires a policy file written in JSON with the following schema: 

## Policy

| Name                                 | Type                   | Required | Description                                        |
|:-------------------------------------|------------------------|----------|----------------------------------------------------|
| instance_min_count                   | int                    | true     |minimal number of instance count                    |
| instance_max_count                   | int                    | true     |maximal number of instance count                    |
| scaling_rules                        | JSON Array<scaling_rules>   | `AnyOf`  |dynamic scaling rules, see `Scaling Rules ` below   |
| schedules                            | JSON Array<schedules>       | `AnyOf`  |scheduled, see `Schedules` below              |


### Scaling Rules 

| Name                 | Type         | Required|Description                                                                      |
|:---------------------|--------------|---------|---------------------------------------------------------------------------------|
| metric_type          | String       | true    |one of the following metric types:memoryused,memoryutil,responsetime, throughput, cpu|| threshold            | int          | true    |the boundary when metric value exceeds is considered as a breach                 |
| operator             | String       | true    |>, <, >=, <=                                                                     |
| adjustment           | String       | true    |the adjustment approach for instance count with each scaling.  Support regex format `^[-+][1-9]+[0-9]*[%]?$`, i.e. +5 means adding 5 instances, -50% means shrinking to the half of current size.  |
| breach_duration_secs | int, seconds | false   |time duration to fire scaling event if it keeps breaching                        |
| cool_down_secs       | int,seconds  | false   |the time duration to wait before the next scaling kicks in                       |


### Schedules

| Name                                 | Type                      | Required|Description                                     |
|:-------------------------------------|---------------------------|---------|------------------------------------------------|
| timezone                             | String                    | true    |Using [timezone definition of Java][a]          |
| recurring_schedule                   | JSON Array<recurring_schedules>| `AnyOf`   |the schedules which will take effect repeatly, see `Recurring Schedule` below |
| specific_date                        | JSON Array<specific_date>      | `AnyOf`   |the schedules which take effect only once, see `Specific Date` below     |

#### Recurring Schedule 

| Name                                 | Type                | Required| Description                                                                             |
|:-------------------------------------|---------------------|---------|-----------------------------------------------------------------------------------------|
| start_date                           | String,"yyyy-mm-dd" | false   | the start date of the schedule. Must be a future time .                                 |
| end_date                             | String,"yyyy-mm-dd" | false   | the end date of the schedule. Must be a future time.                                    |
| start_time                           | String,"hh:mm"      | true    | the start time of the schedule                                                          |
| end_time                             | String,"hh:mm"      | true    | the end time of the schedule                                                            |
| days_of_week / days_of_month         | Array<int>          | false   | recurring days of a week or month. Use [1,2,..,7] or [1,2,...,31] to define it          |
| instance_min_count                   | int                 | true    | minimal number of instance count for this schedule                                      |
| instance_max_count                   | int                 | true    | maximal number of instance count for this schedule                                      |
| initial_min_instance_count           | int                 | false   | the initial minimal number of instance count for this schedule                          |

#### Specific Date 

| Name                                 | Type                       | Required| Description                                                                |
|:-------------------------------------|----------------------------|---------|----------------------------------------------------------------------------|
| start_date_time                      | String,"yyyy-mm-ddThh:mm"  | true    | the start time of the schedule. Must be a future time                      |
| start_date_time                      | String,"yyyy-mm-ddThh:mm"  | true    | the end time of the schedule. Must be a future time                        |
| instance_min_count                   | int                        | true    | minimal number of instance count for this schedule                         |
| instance_max_count                   | int                        | true    | maximal number of instance count for this schedule                         |
| initial_min_instance_count           | int                        | false   | the initial minimal number of instance count for this schedule             |

## Constraints

* If one schedule overlaps another, the one which **starts** first will be guaranteed, while the later one is completely ignored. For example: 

    - Schedule #1:  --------sssssssssss---------------------------- 
    - Schedule #2:  ---------------ssssssssssssss-----------------
    - Schedule #3:  --------------------------sssssssss------------     

    With above definition, schedule #1 and #3 will be applied, while scheudle #2 is ignored.

* If a schedule's start time is earlier than the policy creation/update time, the schedule will not be executed. For example: 

    - Schedule #1:  09:00 - 13:00 , Everyday
   
    If above schedule is created at 10:00AM someday, it won't take effect when it creates, but will be certainly triggered on the next day.  

## Sample Policy

* [Autoscaling policy with dynamic scaling rules][policy-dynamic]
* [Autoscaling policy with dynamic scaling rules and schedules][policy-all]


[a]:https://docs.oracle.com/javase/8/docs/api/java/util/TimeZone.html
[policy-dynamic]: /app-autoscaler/dynamicpolicy.json
[policy-all]: /app-autoscaler/fullpolicy.json

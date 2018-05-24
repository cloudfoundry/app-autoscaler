App-AutoScaler Policy Definition 
================================

**Policy Definition:**

+--------------------------------------+------------------------+---------+----------------------------------------------------+
| Name                                 | Type                   | Required|Description                                         |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| instance_min_count                   | int                    | true    |minimal number of instance count                    |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| instance_max_count                   | int                    | true    |maximal number of instance count                    |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| scaling_rules                        | Array<scaling_rules>   | AnyOf   |dynamic scaling rules                               |
+--------------------------------------+------------------------+ the two +----------------------------------------------------+
| schedules                            | Array<schedules>       |         |schedule definition                                 |
+--------------------------------------+------------------------+---------+----------------------------------------------------+


**Dynamic Scaling Rules Definition "scaling_rules" (part of the "Policy" configuration) :**

+--------------------------------------+------------------------+---------+----------------------------------------------------+
| Name                                 | Type                   | Required|Description                                         |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| metric_type                          | String                 | true    |one of the support metric types:                    |
|                                      |                        |         |memoryused,memoryutil,responsetime, throughput      |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| stat_window_secs                     | int, seconds           | false   |interval to take the avergae metric statistic       |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| breach_duration_secs                 | int, seconds           | false   |interval to fire scaling event if keeping breach    |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| threshold                            | int                    | true    |the number to be breached                           |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| operator                             | String                 | true    |>, <, >=, <=                                        |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| adjustment                           | int                    | true    |the adjustment for instance count with each scaling |
+--------------------------------------+------------------------+---------+----------------------------------------------------+
| cool_down_secs                       | int,seconds            | false   |minimal waiting interval between 2 scaling events   |
+--------------------------------------+------------------------+---------+----------------------------------------------------+


**Schedule Definition "schedules" (part of the "Policy" configuration) :**

+--------------------------------------+---------------------------+---------+-----------------------------------------------------------------+
| Name                                 | Type                      | Required|Description                                                      |
+--------------------------------------+---------------------------+---------+-----------------------------------------------------------------+
| timezone                             | String                    | true    |Using timezone definition of Java.                               |
|                                      |                           |         |https://docs.oracle.com/javase/8/docs/api/java/util/TimeZone.html|
+--------------------------------------+---------------------------+---------+-----------------------------------------------------------------+
| recurring_schedule                   | Array<recurring_schedules>| AnyOf   |the schedules which will take effect repeatly                    |
+--------------------------------------+---------------------------+ the two +-----------------------------------------------------------------+
| specific_date                        | Array<specific_date>      |         |the schedules which take effect only once                        |
+--------------------------------------+---------------------------+---------+-----------------------------------------------------------------+

**Recurring Schedule Definition "recurring_schedule" (part of the "schedules" configuration) :**

+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| Name                                 | Type                | Required| Description                                                                             |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| start_date                           | String,"yyyy-mm-dd" | false   | the start date of the schedule. Must be a future time greater than "NOW".               |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| end_date                             | String,"yyyy-mm-dd" | false   | the end date of the schedule. Must be a future time greater than "NOW".                 |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| start_time                           | String,"hh:mm"      | true    | the start time of the schedule                                                          |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| end_time                             | String,"hh:mm"      | true    | the end time of the schedule                                                            |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| days_of_week                         | Array<int>          | Exactly | recurrence frequency. Use [1,2,..,7] to define the day of week                          |
+--------------------------------------+---------------------+ one of  +-----------------------------------------------------------------------------------------+
| days_of_month                        | Array<int>          | the two | recurrence frequency. Use [1,2,...,31] to define the day of month                       |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| instance_min_count                   | int                 | true    | minimal number of instance count for this schedule                                      |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| instance_max_count                   | int                 | true    | maximal number of instance count for this schedule                                      |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| initial_min_instance_count           | int                 | false   | the initial minimal number of instance count for this schedule                          |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+

**Specific Date Definition "specific_date" (part of the "schedules" configuration) :**

+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| Name                                 | Type                       | Required| Description                                                                |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| start_date_time                      | String,"yyyy-mm-ddThh:mm"  | true    | the start time of the schedule. Must be a future time greater than "NOW".  |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| start_date_time                      | String,"yyyy-mm-ddThh:mm"  | true    | the end time of the schedule. Must be a future time greater than "NOW".    |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| instance_min_count                   | int                        | true    | minimal number of instance count for this schedule                         |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| instance_max_count                   | int                        | true    | maximal number of instance count for this schedule                         |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| initial_min_instance_count           | int                        | false   | the initial minimal number of instance count for this schedule             |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+

**Note**

You need to aware the following facts when define multiple schedules in policy:

* If one schedule overlaps another, the one which starts first will be guaranteed, while the later one is completely ignored. For example: 

+-----+---------------------------------------------------------------------------------------+
|Index|Schedule                                                                               |
+-----+---------------------------------------------------------------------------------------+
|1    |8:00AM ~ 8:00PM every Tuesday , with instance_min_count=1, instance_max_count=10       |
+-----+---------------------------------------------------------------------------------------+
|2    |10:00AM ~ 10:00PM on 2019-01-01, with instance_min_count=5, instance_max_count=50      |
+-----+---------------------------------------------------------------------------------------+

With above definition, on the day of 2019-01-01 (which is Tuesday), schedule #1 will be executed as it occurs first, and schedule #2 will be discarded. 

* If a schedule's start time is earlier than the policy creation/update time, the schedule will not be executed. For example: 

+-----+---------------------------------------------------------------------------------------+
|Index|Schedule                                                                               |
+-----+---------------------------------------------------------------------------------------+
|1    |8:00AM ~ 8:00PM every Tuesday , with instance_min_count=1, instance_max_count=10       |
+-----+---------------------------------------------------------------------------------------+

You may create above schedule at 9:00AM 2019-01-01 (which is Tuesday). Then, it won't take effect on the day of 2019-01-01, but will be certainly triggered on the next Tuesday. 

**Reference**

`Sample policy <https://github.com/cloudfoundry-incubator/app-autoscaler/blob/develop/src/integration/fakePolicyWithSchedule.json>`_
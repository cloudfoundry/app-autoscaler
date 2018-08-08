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
| schedules                            | Array<schedules>       |         |scheduled scaling rules                             |
+--------------------------------------+------------------------+---------+----------------------------------------------------+


**Dynamic Scaling Rules Definition "scaling_rules" (part of the "Policy" configuration) :**

+--------------------------------------+------------------------+---------+----------------------------------------------------------------+
| Name                                 | Type                   | Required|Description                                                     |
+--------------------------------------+------------------------+---------+----------------------------------------------------------------+
| metric_type                          | String                 | true    |one of the following metric types:                              |
|                                      |                        |         |memoryused,memoryutil,responsetime, throughput                  |
+--------------------------------------+------------------------+---------+----------------------------------------------------------------+
| breach_duration_secs                 | int, seconds           | false   |time duration to fire scaling event if it keeps breaching       |
+--------------------------------------+------------------------+---------+----------------------------------------------------------------+
| threshold                            | int                    | true    |the boundary when metric value exceeds is considered as a breach|
+--------------------------------------+------------------------+---------+----------------------------------------------------------------+
| operator                             | String                 | true    |>, <, >=, <=                                                    |
+--------------------------------------+------------------------+---------+----------------------------------------------------------------+
| adjustment                           | int                    | true    |the adjustment for instance count with each scaling             |
+--------------------------------------+------------------------+---------+----------------------------------------------------------------+
| cool_down_secs                       | int,seconds            | false   |the time duration to wait before the next scaling kicks in      |
+--------------------------------------+------------------------+---------+----------------------------------------------------------------+


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
| start_date                           | String,"yyyy-mm-dd" | false   | the start date of the schedule. Must be a future time .                                 |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| end_date                             | String,"yyyy-mm-dd" | false   | the end date of the schedule. Must be a future time.                                    |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| start_time                           | String,"hh:mm"      | true    | the start time of the schedule                                                          |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| end_time                             | String,"hh:mm"      | true    | the end time of the schedule                                                            |
+--------------------------------------+---------------------+---------+-----------------------------------------------------------------------------------------+
| days_of_week                         | Array<int>          | Exactly | recurring days of a week. Use [1,2,..,7] to define it                                   |
+--------------------------------------+---------------------+ one of  +-----------------------------------------------------------------------------------------+
| days_of_month                        | Array<int>          | the two | recurring days of a month . Use [1,2,...,31] to define it                               |
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
| start_date_time                      | String,"yyyy-mm-ddThh:mm"  | true    | the start time of the schedule. Must be a future time                      |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| start_date_time                      | String,"yyyy-mm-ddThh:mm"  | true    | the end time of the schedule. Must be a future time                        |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| instance_min_count                   | int                        | true    | minimal number of instance count for this schedule                         |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| instance_max_count                   | int                        | true    | maximal number of instance count for this schedule                         |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+
| initial_min_instance_count           | int                        | false   | the initial minimal number of instance count for this schedule             |
+--------------------------------------+----------------------------+---------+----------------------------------------------------------------------------+

**Constraints**

* If one schedule overlaps another, the one which **starts** first will be guaranteed, while the later one is completely ignored. For example: 

    - Schedule #1:  --------sssssssssss---------------------------- 
    - Schedule #2:  ---------------ssssssssssssss-----------------
    - Schedule #3:  --------------------------sssssssss------------     

    With above definition, schedule #1 and #3 will be applied, while scheudle #2 is ignored.

* If a schedule's start time is earlier than the policy creation/update time, the schedule will not be executed. For example: 

    - Schedule #1:  09:00 - 13:00 , Everyday
   
    If above schedule is created at 10:00AM someday, it won't take effect when it creates, but will be certainly triggered on the next day.  

**Reference**

`Sample policy <https://github.com/cloudfoundry-incubator/app-autoscaler/blob/develop/src/integration/fakePolicyWithSchedule.json>`_
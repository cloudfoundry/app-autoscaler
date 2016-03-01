==================================================
AutoScaler REST API Specification
==================================================

.. contents:: Table of Contents

Overview
========
The Auto-Scaler API server provides a set of REST APIs to manage scaling policies, retrieve the scaling history and metric data.

This document provides the specification of the APIs.  See `How to use AutoScaler REST APIs`_ For the usage of these APIs. 

.. _`How to use AutoScaler REST APIs`: API_usage.rst 

Policy Management APIs: 
============================================
Policy are configurations of scaling rules that Autoscaler service used to specify the conditions in which certain scaling activities are triggered. 

Create/Update Policy: ``PUT /v1/autoscaler/apps/<app_id>/policy/``
--------------------------------
Create a new Auto-scaling policy or update the existing policy for your application. In this API, you need to supply a policy in JSON format.

* Request: ``PUT /v1/autoscaler/apps/<app_id>/policy/``

=============== ======================================================
Request         ``PUT /v1/autoscaler/apps/<app_id>/policy/``
Authorization   Authorization: Bearer Token of ``AccessToken`` 
Headers         ``Content-Type: application/json`` and ``Accept: application/json``
Parameters      app_id: the GUID of the application
Request Body    *example* ::

                    {
                        "instanceMinCount":1,
                        "instanceMaxCount":5,
                        "policyTriggers":[{
                            "metricType":"Memory",
                            "statWindow":300,
                            "lowerThreshold":30,
                            "upperThreshold":80,
                            "instanceStepCountDown":1,
                            "instanceStepCountUp":1,
                            "stepDownCoolDownSecs":600,
                            "stepUpCoolDownSecs":600
                        }],
                        "schedules": {
                            "timezone":"(GMT +08:00) Asia/Shanghai",
                            "recurringSchedule":[{
                              "startTime":"00:00",
                              "endTime":"23:59",
                              "repeatOn":"[\"1\",\"2\",\"3\",\"4\",\"5\"]",
                              "minInstCount":2,
                              "maxInstCount":5
                              }],
                            "specificDate":[{
                               "startDate":"2017-06-19",
                               "startTime":"00:00",
                               "endDate":"2017-06-19",
                               "endTime":"23:59",
                               "minInstCount":1,
                               "maxInstCount":5
                            }]
                         }
                    }


Response Codes   *Codes* ::

                    201 - Created 
                    200 - Updated 
                    400 - Bad Request
                    401 - Unauthorized
                    404 - Not Found - app_id not found
 
Response Body     Successful Response

                  {
                    "policyId": "5281374a-ae80-4205-8123-6cafcc326514"
                  }
                  
                  Error Response

                  {
                    "error": "error message" 
                  }

=============== ======================================================


Fields            *Available Fields* ::

                    instanceMinCount -  int              -  Required - minimal number of instance count
                    instanceMaxCount -  int              -  Required - maximal number of instance count
                    policyTriggers   -  List<Trigger>    -  Required - Trigger setting for this policy, each trigger is defined as below 
                     metricType             -  String -  Required - enumerated type for this trigger, see Appendix for currently available values
                     statWindow             -  int    -  Optional - interval to calculate statistics in seconds 
                     breachDuration         -  int    -  Optional - breach duration in seconds is divided by the periodicity of the metric to determine the number of data points that will result in a scaling event
                     lowerThreshold         -  int    -  Optional - lower threshold in percentage that will trigger a scaling event, usually scaling-in
                     upperThreshold         -  int    -  Optional - upper threshold in percentage that will trigger a scaling event, usually scaling-out
                     instanceStepCountDown  -  int    -  Optional - number of instances to reduce per scaling-in action
                     instanceStepCountUp    -  int    -  Optional - number of instances to increase per scaling-out action
                     stepDownCoolDownSecs   -  int    -  Optional - cool down period that prevent further scaling in action
                     stepUpCoolDownSecs     -  int    -  Optional - cool down period that prevent further scaling out  action 
                     schedules       -  Object           -  Optional  - Schedule based scaling settings  defined as below    
                       recurringSchedule  - List<recurringSchedule>   -  Optional  - list of recurring schedules,  defined as below
                         minInstCount  - int        -  Required  - minimal instance count in this rule
                         maxInstCount  - int        -  Optional  - maximal  instance count in this rule
                         startTime     - String     -  Required  - start time for this rule to take effect
                         endTime       - String     -  Required  - end time for this rule to take effect
                         repeatOn      - String     -  Required  - day of week that the rule to take effect
                       specificDate  - List<specificDate>             -  Optional  - List of specific dates, defined as below
                         minInstCount  - int        -  Required  - minimal instance count in this rule
                         maxInstCount  - int        -  Optional  - maximal  instance count in this rule
                         startDate     - String     -  Required  - start date for this rule to take effect
                         startTime     - String     -  Required  - start time for this rule to take effect
                         endDate       - String     -  Required  - end date for this rule to take effect
                         endTime       - String     -  Required  - end time for this rule to take effect
                     timezone        - String                         -  Optional   - Timezone setting for the dates/times (See Appendix for available values)


Delete Policy: ``DELETE /v1/autoscaler/apps/:app_id/policy/``
--------------------------------
Delete the existing policy from your application

* Request: ``DELETE /v1/autoscaler/apps/<app_id>/policy/``

=============== =================================================
Request         ``DELETE /v1/autoscaler/apps/:app_id/policy/``
Authorization   Authorization: Bearer Token of ``AccessToken`` 
Headers         ``Accept: application/json``
Parameters      app_id: the GUID of application
Request Body    None
Response Codes   *Codes* ::

                    200 - Success
                    400 - Bad Request
                    401 - Unauthorized
                    404 - Not Found - app_id not found or no policy found associated with this applicaiton
 
Response Body     Successful Response

                  {
                    
                  }
                  
                  Error Response

                  {
                    "error": "error message" 
                  }

=============== =================================================

Get Policy: ``GET /v1/autoscaler/apps/<app_id>/policy/``
--------------------------------
Get existing policy of your application

* Request: ``GET /v1/autoscaler/apps/<app_id>/policy/``

=============== ====================================================================
Request         ``GET /v1/autoscaler/apps/<app_id>/policy/``
Authorization   Authorization: Bearer Token of ``AccessToken``
Headers         ``Accept: application/json``
Parameters      app_id: the GUID of the application
Request Body    none

Response Codes   *Codes* ::

                    200 - Success
                    400 - Bad Request
                    401 - Unauthorized
                    404 - Not Found - app_id not found
 
Response Body     Successful Response

                    {
                        "policyState": ENABLED,
                        "instanceMinCount":1,
                        "instanceMaxCount":5,
                        "policyTriggers":[{
                            "metricType":"Memory",
                            "statWindow":300,
                            "lowerThreshold":30,
                            "upperThreshold":80,
                            "instanceStepCountDown":1,
                            "instanceStepCountUp":1,
                            "stepDownCoolDownSecs":600,
                            "stepUpCoolDownSecs":600
                                "wantAssertionSigned": false
                        }],
                        "schedules": {
                            "timezone":"(GMT +08:00) Asia/Shanghai",
                            "recurringSchedule":[{
                              "startTime":"00:00",
                              "endTime":"23:59",
                              "repeatOn":"[\"1\",\"2\",\"3\",\"4\",\"5\"]",
                              "minInstCount":2,
                              "maxInstCount":5
                              }],
                            "specificDate":[{
                               "startDate":"2017-06-19",
                               "startTime":"00:00",
                               "endDate":"2017-06-19",
                               "endTime":"23:59",
                               "minInstCount":1,
                               "maxInstCount":5
                            }]
                         }
                    }

                  
                  Error Response

                  {
                    "error": "error message" 
                  }

=============== ====================================================================


Fields            *Available Fields* ::

                    policyState      -  String           - Current policy status, ENABLED or DISABLED
                    instanceMinCount -  int              - minimal number of instance count
                    instanceMaxCount -  int              - maximal number of instance count
                    policyTriggers   -  List<Trigger>    - Trigger setting for this policy, each trigger is defined as below 
                         metricType              -  String    - enumerated type for this trigger, see Appendix for currently available values
                         statWindow              -  int       - time interval in seconds for metric value statistics 
                         breachDuration          -  int       - breach duration in seconds is divided by the periodicity of the metric to determine the number of data points that will result in a scaling event
                         lowerThreshold          -  int       - lower threshold in percentage that will trigger a scaling event, usually scaling-in
                         upperThreshold          -  int       - upper threshold in percentage that will trigger a scaling event, usually scaling-out
                         instanceStepCountDown   -  int       - number of instance to reduce per scaling-in action
                         instanceStepCountUp     -  int       - number of instance to increase per scaling-out action
                         stepDownCoolDownSecs    -  int       - cool down period that prevent further scaling in action
                         stepUpCoolDownSecs      -  int       - cool down period that prevent further scaling out action 
                     schedules       -  Object           - schedule based scaling settings  defined in below    
                          recurringSchedule  - List<recurringSchedule> - list of recurring schedules, defined as below
                                  minInstCount  - int        - minimal instance count in this rule
                                  maxInstCount  - int        - maximal  instance count in this rule
                                  startTime     - String     - start time for the rule to take effect
                                  endTime       - String     - end time for the rule to take effect
                                  repeatOn      - String     - days of week for the rule to take effect
                         specificDate        - List<specificDate>      - List of the specific dates, defined as below
                                  minInstCount  - int        - minimal instance count in this rule
                                  maxInstCount  - int        - maxmal  instance count in this rule
                                  startDate     - String     - start date for this rule to take effect
                                  startTime     - String     - start time for this rule to take effect
                                  endDate       - String     - end date for this rule to take effect
                                  endTime       - String     - end time for this rule to take effect
                         timezone            - String                  - Timezone setting for dates/times  (See Appendix for available values)


Policy Status APIs: 
============================================
You can use these APIs to enable/disable the policy or check current policy status

Enable/Disable policy: ``PUT /v1/autoscaler/apps/:app_id/policy/status/``
--------------------------------
Enable the suspended policy or disable the policy temporarily

* Request: ``PUT /v1/autoscaler/apps/:app_id/policy/status/``

=============== =================================================
Request         ``PUT /v1/autoscaler/apps/:app_id/policy/status/``
Authorization   Authorization: Bearer Token of ``AccessToken``
Headers         ``Accept: application/json``
Parameters      app_id: the GUID of the application
Request Body    None
Response Codes   *Codes* ::

                    200 - Success
                    400 - Bad Request
                    401 - Unauthorized
                    404 - Not Found - app_id not found or policy not found
 
Response Body     Successful Response

                  {
                    
                  }
                  
                  Error Response

                  {
                    "error": "error message" 
                  }

=============== =================================================

Get policy status: ``GET /v1/autoscaler/apps/:app_id/policy/status/``
--------------------------------
Get the policy status of your application

* Request: ``GET /v1/autoscaler/apps/:app_id/policy/status/``

=============== =================================================
Request         ``GET /v1/autoscaler/apps/:app_id/policy/status/``
Authorization   Authorization: Bearer Token of ``AccessToken``
Headers         ``Accept: application/json``
Parameters      app_id: the GUID of application
Request Body    None
Response Codes   *Codes* ::

                    200 - Success
                    400 - Bad Request
                    401 - Unauthorized
                    404 - Not Found - app_id not found or policy not found
 
Response Body     Successful Response

                  {
                    "status": "ENABLED"
                  }
                  
                  Error Response

                  {
                    "error": "error message" 
                  }
Fields            *Available Fields* ::

                    status      -  String           -  Required - Current policy status, ENABLED or DISABLED
=============== =================================================

Scaling Data Management APIs: 
============================================
You can use these APIs to retrieve scaling history and metric data

Get autoscaling history: ``GET /v1/autoscaler/apps/<app_id>/scalinghistory/?startTime={startTime}&endTime={endTime}``
--------------------------------
Get scaling history of your application

* Request: ``GET /v1/autoscaler/apps/<app_id>/scalinghistory/?startTime={startTime}&endTime={endTime}``

=============== =================================================
Request         ``GET /v1/autoscaler/apps/<app_id>/scalinghistory/?startTime={startTime}&endTime={endTime}``
Authorization   Authorization: Bearer Token of ``AccessToken``
Headers         ``Accept: application/json``
Parameters      app_id: the GUID of the application. startTime and endTime are the timestamp in millisecond to specify the time range of scaling action 
Request Body    None
Response Codes   *Codes* ::

                    200 - Success
                    400 - Bad Request
                    401 - Unauthorized
                    404 - Not Found - app_id not found or policy not found
 
Response Body     Successful Response

                  {
                     "data":[
                         {
                                 "appId":"b56d1c9c-45e5-44b2-9f12-68684ccdd545",
                                 "status":”COMPLETE”,
                                 "instancesBefore":2,
                                 "instancesAfter":5,
                                 "startTime":1434971992115,
                                 "endTime":0,
                                  "trigger":{
                                         "metrics":"Memory",
                                         "threshold":10,
                                         "thresholdType":"upper",
                                         "breachDuration":30,
                                         "triggerType":”PolicyChange”
                                 },
                                 "errorCode":null
                         },{
                                 "appId":"b56d1c9c-45e5-44b2-9f12-68684ccdd545",
                                 "status":”FAILED”,
                                 "instancesBefore":1,
                                 "instancesAfter":1,
                                 "startTime":1434971992479,
                                 "endTime":1434971992484,
                                 "trigger":{
                                          "metrics":"Memory",
                                          "threshold":10,
                                          "thresholdType":"upper",
                                          "breachDuration":30,
                                          "triggerType":”MonitorEvent”
                                 },
                                 "errorCode":"CF-AppMemoryQuotaExceeded"
                        }
                     ],
                     "timestamp": 0
                  }
                  
                  Error Response

                  {
                    "error": "error message" 
                  }
Fields            *Available Fields* ::

                     data          -  List<HistoryData>   - List of scaling history data, see below for its structure
                         appId       -  String           - App GUID  
                         status      -  String           - enumerated Scaling state, e.g. `FAILED` `REALIZING` `COMPLETE`
                         instanceBefore -  int           - instance count before the scaling action
                         instancesAfter -  int           - instance count after the scaling action
                         startTime      -  long          - time stamp of scaling start time
                         endTime        -  long          - time stamp of scaling end time
                         errorCode      -  String        - error code of failed scaling, say `CloudFoundryInternalError`
                         trigger        -  Object        - the event that trigger this scaling, see below for its structure
                                 metrics         -  String     - metric type of the scaling event 
                                 threshold       -  int        - threshold of the scaling event 
                                 thresholdType   - String      - threshold type of the scaling event, e.g. `lower` `upper`
                                 breachDuration  - int         - breach duration of the scaling event
                                 triggerType     -  String     - trigger type of the scaling event, e.g. `PolicyChange` `MonitorEvent`
                     timestamp     -  long                - start time in millisecond of last history data when not all data are returned, 0 if all data returned for this request

=============== =================================================

Get Scaling metrics: ``GET /v1/autoscaler/apps/:app_id/metrics?startTime={startTime}&endTime={endTime}``
--------------------------------
Get scaling metric data

* Request: ``GET /v1/autoscaler/apps/:app_id/metrics?startTime={startTime}&endTime={endTime}``

=============== =================================================
Request         ``GET /v1/autoscaler/apps/:app_id/metrics?startTime={startTime}&endTime={endTime}``
Authorization   Authorization: Bearer Token of ``AccessToken``
Headers         ``Accept: application/json``
Parameters      app_id: the GUID of the application time stamp in millisecond to specify the time range of metric data 
Request Body    None
Response Codes   *Codes* ::

                    200 - Success
                    400 - Bad Request
                    401 - Unauthorized
                    404 - Not Found - app_id not found or policy not found
 
Response Body     Successful Response
                  {
                    "data":
                    [
                         {
                                 "appId": "5291be1d-01b5-4114-bc0a-d21fdf175b6a",
                                 "appName": "Hello",
                                 "appType": "java",
                                 "timestamp": 1456551515225,
                                 "instanceMetrics":
                                 [
                                           {
                                                 "instanceIndex": 0,
                                                 "timestamp": 1456551515223,
                                                 "instanceId": "0",
                                                 "metrics":
                                                 [
                                                     {
                                                         "name": "Memory",
                                                         "value": "176.02734375",
                                                         "category": "cf-stats",
                                                         "group": "Memory",
                                                         "timestamp": 1456551515000,
                                                         "unit": "MB",
                                                         "desc": null
                                                    },
                                                    {
                                                         "name": "CPU",
                                                         "value": "0.8597572518277989",
                                                         "category": "cf-stats",
                                                         "group": "CPU",
                                                         "timestamp": 1456551515000,
                                                         "unit": "%",
                                                         "desc": null
                                                    }
                                                 ]
                                           }
                                 ]
                         },
                         {
                                 "appId": "5291be1d-01b5-4114-bc0a-d21fdf175b6a",
                                 "appName": "Hello",
                                 "appType": "java",
                                 "timestamp": 1456551635267,
                                 "instanceMetrics":
                                 [
                                          {
                                                 "instanceIndex": 0,
                                                 "timestamp": 1456551635267,
                                                 "instanceId": "0",
                                                  "metrics":
                                                 [
                                                    {
                                                         "name": "Memory",
                                                         "value": "176.0625",
                                                         "category": "cf-stats",
                                                         "group": "Memory",
                                                         "timestamp": 1456551635000,
                                                         "unit": "MB",
                                                         "desc": null
                                                   },
                                                   {
                                                         "name": "CPU",
                                                         "value": "0.24369401867181975",
                                                         "category": "cf-stats",
                                                         "group": "CPU",
                                                         "timestamp": 1456551635000,
                                                         "unit": "%",
                                                         "desc": null
                                                   }
                                                ]
                                            }
                                ]
                        }
                   ],
                   "timestamp": 0
                }
                  Error Response

                  {
                    "error": "error message" 
                  }
Fields            *Available Fields* ::

                    data      -  List<AppInstanceMetrics>           -  metrics data ordered by time stamp, see below for its structure
                         appId     -  String           -  application  GUID 
                         appName   -  String           -  application  name
                         appType   -  String           -  type of the application, see Appendix for currently available value
                         InstanceMetrics  - List<InstanceMetrics> - detailed instance metric data, see below for its structure
                                 InstanceIndex      - int       - index of the application instance
                                 timeStamp          - long      - time that the metric data is collected
                                 InstanceId         - String    - ID of the application instance
                                 Metrics            - List<Metric>   - specific metric data, see below for its structure
                                         name       - String        - metric name
                                         value      - float         - metric value 
                                         category   - String        - category that this metric belongs to
                                         group      - String        - group name that this metric belongs to 
                                         timestamp  - String        - time stamp of this data collected
                                         unit       - String        - unit of metric data
                                         desc       - String        - description of this metric
                   timeStamp  - String                - time stamp of last metric data when not all data returned, 0 if all data returned for this request
=============== =================================================

Appendix
===================

Metric Type values
--------------------------------
Currently the following values are supported:
    “Memory”
    
Trigger Type values
--------------------------------
Currently the following values are supported:
    “PolicyChange”, “MonitorEvent”
    
Scaling status values
--------------------------------
Currently the following values are supported:
    “READY”, “REALIZING”, “COMPLETED”, “FAILED”
    
Application type values
--------------------------------
Currently the following values are supported:
    "java", "ruby", "ruby_sinatra", "ruby_on_rails", "nodejs", "go", "php", "python", "dotnet", "unknown"
    
Threshold type values
--------------------------------
Currently the following values are supported:
   "upper", "lower" or “”
    
Available TimeZone values
--------------------------------
Currently the following values are supported:
::

    "(GMT +08:00) Asia/Chongqing",
    "(GMT +08:00) Asia/Chungking",
    "(GMT +08:00) Asia/Harbin",
    "(GMT +08:00) Asia/Hong_Kong",
    "(GMT +08:00) Asia/Irkutsk",
    "(GMT +08:00) Asia/Kuala_Lumpur",
    "(GMT +08:00) Asia/Kuching",
    "(GMT +08:00) Asia/Macao",
    "(GMT +08:00) Asia/Macau",
    "(GMT +08:00) Asia/Makassar",
    "(GMT +08:00) Asia/Manila",
    "(GMT +08:00) Asia/Shanghai",
    "(GMT +08:00) Asia/Singapore",
    "(GMT +08:00) Asia/Taipei",
    "(GMT +08:00) Asia/Ujung_Pandang",
    "(GMT +08:00) Asia/Ulaanbaatar",
    "(GMT +08:00) Asia/Ulan_Bator",
    "(GMT +08:00) Australia/Perth",
    "(GMT +08:00) Australia/West",
    "(GMT +08:00) Etc/GMT-8",
    "(GMT +08:00) Hongkong",
    "(GMT +08:00) PRC",
    "(GMT +08:00) ROC",
    "(GMT +08:00) Singapore",
    "(GMT +08:45) Australia/Eucla",
    "(GMT +09:00) Asia/Dili",
    "(GMT +09:00) Asia/Jayapura",
    "(GMT +09:00) Asia/Khandyga",
    "(GMT +09:00) Asia/Pyongyang",
    "(GMT +09:00) Asia/Seoul",
    "(GMT +09:00) Asia/Tokyo",
    "(GMT +09:00) Asia/Yakutsk",
    "(GMT +09:00) Etc/GMT-9",
    "(GMT +09:00) Japan",
    "(GMT +09:00) Pacific/Palau",
    "(GMT +09:00) ROK",
    "(GMT +09:30) Australia/Adelaide ",
    "(GMT +09:30) Australia/Broken_Hill",
    "(GMT +09:30) Australia/Darwin",
    "(GMT +09:30) Australia/North",
    "(GMT +09:30) Australia/South",
    "(GMT +09:30) Australia/Yancowinna ",
    "(GMT +10:00) Antarctica/DumontDUrville",
    "(GMT +10:00) Asia/Magadan",
    "(GMT +10:00) Asia/Sakhalin",
    "(GMT +10:00) Asia/Ust-Nera",
    "(GMT +10:00) Asia/Vladivostok",
    "(GMT +10:00) Australia/ACT",
    "(GMT +10:00) Australia/Brisbane",
    "(GMT +10:00) Australia/Canberra",
    "(GMT +10:00) Australia/Currie",
    "(GMT +10:00) Australia/Hobart",
    "(GMT +10:00) Australia/Lindeman",
    "(GMT +10:00) Australia/Melbourne",
    "(GMT +10:00) Australia/NSW",
    "(GMT +10:00) Australia/Queensland",
    "(GMT +10:00) Australia/Sydney",
    "(GMT +10:00) Australia/Tasmania",
    "(GMT +10:00) Australia/Victoria",
    "(GMT +10:00) Etc/GMT-10",
    "(GMT +10:00) Pacific/Chuuk",
    "(GMT +10:00) Pacific/Guam",
    "(GMT +10:00) Pacific/Port_Moresby",
    "(GMT +10:00) Pacific/Saipan",
    "(GMT +10:00) Pacific/Truk",
    "(GMT +10:00) Pacific/Yap",
    "(GMT +10:30) Australia/LHI",
    "(GMT +10:30) Australia/Lord_Howe",
    "(GMT +11:00) Antarctica/Macquarie",
    "(GMT +11:00) Asia/Srednekolymsk",
    "(GMT +11:00) Etc/GMT-11",
    "(GMT +11:00) Pacific/Bougainville",
    "(GMT +11:00) Pacific/Efate",
    "(GMT +11:00) Pacific/Guadalcanal",
    "(GMT +11:00) Pacific/Kosrae",
    "(GMT +11:00) Pacific/Noumea",
    "(GMT +11:00) Pacific/Pohnpei",
    "(GMT +11:00) Pacific/Ponape",
    "(GMT +11:30) Pacific/Norfolk",
    "(GMT +12:00) Antarctica/McMurdo",
    "(GMT +12:00) Antarctica/South_Pole",
    "(GMT +12:00) Asia/Anadyr",
    "(GMT +12:00) Asia/Kamchatka",
    "(GMT +12:00) Etc/GMT-12",
    "(GMT +12:00) Kwajalein",
    "(GMT +12:00) NZ",
    "(GMT +12:00) Pacific/Auckland",
    "(GMT +12:00) Pacific/Fiji",
    "(GMT +12:00) Pacific/Funafuti",
    "(GMT +12:00) Pacific/Kwajalein",
    "(GMT +12:00) Pacific/Majuro",
    "(GMT +12:00) Pacific/Nauru",
    "(GMT +12:00) Pacific/Tarawa",
    "(GMT +12:00) Pacific/Wake",
    "(GMT +12:00) Pacific/Wallis",
    "(GMT +12:45) NZ-CHAT",
    "(GMT +12:45) Pacific/Chatham",
    "(GMT +13:00) Etc/GMT-13",
    "(GMT +13:00) Pacific/Apia",
    "(GMT +13:00) Pacific/Enderbury",
    "(GMT +13:00) Pacific/Fakaofo",
    "(GMT +13:00) Pacific/Tongatapu",
    "(GMT +14:00) Etc/GMT-14",
    "(GMT +14:00) Pacific/Kiritimati"

  


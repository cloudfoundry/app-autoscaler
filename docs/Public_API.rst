API Server Public Rest API
==========================

Scaling History API
-------------------

**List scaling history of an application**
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

**GET /v1/apps/:guid/scaling\_history**
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

**Request**
^^^^^^^^^^^

Route
'''''

    GET /v1/apps/8d0cee08-23ad-4813-a779-ad8118ea0b91/scaling\_histories

Parameters
''''''''''

+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| Name               | Description                                                                   | Valid values                                                        | Required              | Example values                   |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| guid               | The GUID of the application                                                   |                                                                     | true                  |                                  |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| start-time         | The start time                                                                | int, the number of nanoseconds elapsed since January 1, 1970 UTC.   | false                 | start-time=1494989539138350432   |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| end-time           | The end time                                                                  | int, the number of nanoseconds elapsed since January 1, 1970 UTC.   | false                 | end-time=1494989549117047288     |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| order              | The order type. The scaling history will be order by timestamp asc or desc.   | string,”asc” or “desc”                                              | false. default desc   | order=desc                       |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| page               | The page number to query                                                      | int                                                                 | false                 | page=1                           |
|                    |                                                                               |                                                                     |                       |                                  |
|                    |                                                                               |                                                                     | default               |                                  |
|                    |                                                                               |                                                                     |                       |                                  |
|                    |                                                                               |                                                                     | 1                     |                                  |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| results-per-page   | The number of results per page                                                | int                                                                 | false.                | results-per-page=10              |
|                    |                                                                               |                                                                     |                       |                                  |
|                    |                                                                               |                                                                     | default 10            |                                  |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+

Headers
'''''''

    Authorization: bearer
    eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTQwOCIsImVtYWlsIjoiZW1haWwtMzAzQHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NzF9.RMJZvSzCSxpj4jjZBmzbO7eoSfTAcIWVSHqFu5\_Iu\_o

cURL
''''
    | curl "https://[the-api-server-url]:[port]/v1/apps/8d0cee08-23ad-4813-a779-ad8118ea0b91/scaling\_histories?start-time=1494989539138350432&end-time=1494989539138399999&order=desc&page=1&results-per-page=10" \\
    | -X GET \\
    | -H "Authorization: bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTQwOCIsImVtYWlsIjoiZW1haWwtMzAzQHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NzF9.RMJZvSzCSxpj4jjZBmzbO7eoSfTAcIWVSHqFu5\_Iu\_o" 

Response
^^^^^^^^

Status
''''''

    200 OK

Body
''''

    {

    "total\_results": 2,

    "total\_pages": 1,

    "page": 1,

    "resources": [

    {

    "app\_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",

    "timestamp": 1494989539138350433,

    "scaling\_type": 1,

    "status": 0,

    "old\_instances": 1,

    "new\_instances": 2,

    "reason": "",

    "message": "",

    "error": ""

    },

    {

    "app\_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",

    "timestamp": 1494989539138350435,

    "scaling\_type": 1,

    "status": 0,

    "old\_instances": 1,

    "new\_instances": 2,

    "reason": "",

    "message": "",

    "error": ""

    }

    ]

    }

Application Metric API
----------------------

**List metrics of an application**
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~

**GET /v1/apps/:guid/metrics**
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

**Request**
^^^^^^^^^^^

Route
'''''

    GET /v1/apps/8d0cee08-23ad-4813-a779-ad8118ea0b91/metrics

Parameters
''''''''''

+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| Name               | Description                                                                   | Valid values                                                        | Required              | Example values                   |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| guid               | The GUID of the application                                                   |                                                                     | true                  |                                  |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| metric-type        | The metric type                                                               |                                                                     | true                  | metric-type=memoryused           |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| start-time         | The start time                                                                | int, the number of nanoseconds elapsed since January 1, 1970 UTC.   | true                  | start-time=1494989539138350432   |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| end-time           | The end time                                                                  | int, the number of nanoseconds elapsed since January 1, 1970 UTC.   | true                  | end-time=1494989549117047288     |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| order              | The order type. The scaling history will be order by timestamp asc or desc.   | string,”asc” or “desc”                                              | false. default desc   | order=asc                        |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| page               | The page number to query                                                      | int                                                                 | true                  | page=1                           |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+
| results-per-page   | The number of results per page                                                | int                                                                 | true                  | results-per-page=10              |
+--------------------+-------------------------------------------------------------------------------+---------------------------------------------------------------------+-----------------------+----------------------------------+

Headers
'''''''
    Authorization: bearer
    eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTQwOCIsImVtYWlsIjoiZW1haWwtMzAzQHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NzF9.RMJZvSzCSxpj4jjZBmzbO7eoSfTAcIWVSHqFu5\_Iu\_o

cURL
''''
    | curl "https://[the-api-server-url]:[port]/v1/apps/8d0cee08-23ad-4813-a779-ad8118ea0b91/metrics?start-time=1494989539138350432&end-time=1494989539138399999&metricType=memoryused&orer=asc&page=1&results-per-page=10" \\
    | -X GET \\
    | -H "Authorization: bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTQwOCIsImVtYWlsIjoiZW1haWwtMzAzQHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NzF9.RMJZvSzCSxpj4jjZBmzbO7eoSfTAcIWVSHqFu5\_Iu\_o" 


Response
^^^^^^^^

Status
''''''

    200 OK

Body
''''

    [

    "total\_results": 2,

    "total\_pages": 1,

    "page": 1,

    "resources": [

    {

    "app\_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",

    "instanceIndex": 0,

    "timestamp": 1494989539138350433,

    "collected\_at": 1494989539138350000,

    "metric\_type": "memoryused",

    "value": "400",

    "unit": "megabytes"

    },

    {

    "app\_guid": "8d0cee08-23ad-4813-a779-ad8118ea0b91",

    "instance\_index": 1,

    "timestamp": 1494989539138350433,

    "collected\_at": 1494989539138350000,

    "metric\_type": "memoryused",

    "value": "400",

    "unit": "megabytes"

    }

    ]

    ]

Policy API
----------

Set Policy
~~~~~~~~~~

PUT /v1/apps/:guid/policy
^^^^^^^^^^^^^^^^^^^^^^^^^

Request
^^^^^^^

Route
'''''

    PUT /v1/apps/:guid/policy

Parameters
''''''''''

+--------+-------------------------------+----------------+------------+------------------+
| Name   | Description                   | Valid values   | Required   | Example values   |
+--------+-------------------------------+----------------+------------+------------------+
| guid   | The GUID of the application   |                | true       |                  |
+--------+-------------------------------+----------------+------------+------------------+

Body
''''

    {

    "instance\_min\_count": 1,

    "instance\_max\_count": 4,

    "name": "autoscaler policy"

    "scaling\_rules": [

    {

    "metric\_type": "memoryused",

    "stat\_window\_secs": 300,

    "breach\_duration\_secs": 600,

    "threshold": 30,

    "operator": "<",

    "cool\_down\_secs": 300,

    "adjustment": "-1"

    },

    {

    "metric\_type": "memoryused",

    "stat\_window\_secs": 300,

    "breach\_duration\_secs": 600,

    "threshold": 90,

    "operator": ">=",

    "cool\_down\_secs": 300,

    "adjustment": "+1"

    }

    ],

    "schedules": {

    "timezone": "Asia/Shanghai",

    "recurring\_schedule": [

    {

    "start\_time": "10:00",

    "end\_time": "18:00",

    "days\_of\_week": [

    1,

    2,

    3

    ],

    "instance\_min\_count": 1,

    "instance\_max\_count": 10,

    "initial\_min\_instance\_count": 5

    },

    {

    "start\_date": "2016-06-27",

    "end\_date": "2016-07-23",

    "start\_time": "11:00",

    "end\_time": "19:30",

    "days\_of\_month": [

    5,

    15,

    25

    ],

    "instance\_min\_count": 3,

    "instance\_max\_count": 10,

    "initial\_min\_instance\_count": 5

    },

    {

    "start\_time": "10:00",

    "end\_time": "18:00",

    "days\_of\_week": [

    4,

    5,

    6

    ],

    "instance\_min\_count": 1,

    "instance\_max\_count": 10

    },

    {

    "start\_time": "11:00",

    "end\_time": "19:30",

    "days\_of\_month": [

    10,

    20,

    30

    ],

    "instance\_min\_count": 1,

    "instance\_max\_count": 10

    }

    ],

    "specific\_date": [

    {

    "start\_date\_time": "2015-06-02T10:00",

    "end\_date\_time": "2015-06-15T13:59",

    "instance\_min\_count": 1,

    "instance\_max\_count": 4,

    "initial\_min\_instance\_count": 2

    },

    {

    "start\_date\_time": "2015-01-04T20:00",

    "end\_date\_time": "2015-02-19T23:15",

    "instance\_min\_count": 2,

    "instance\_max\_count": 5,

    "initial\_min\_instance\_count": 3

    }

    ]

    }

    }

Headers
'''''''
Authorization: bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTQwOCIsImVtYWlsIjoiZW1haWwtMzAzQHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NzF9.RMJZvSzCSxpj4jjZBmzbO7eoSfTAcIWVSHqFu5\_Iu\_o

cURL
''''
    | curl
      "https://[the-api-server-url]:[port]/v1/apps/8d0cee08-23ad-4813-a779-ad8118ea0b91/policy" \\
    | -d @policy.json \\
    | -X POST \\
    | -H "Authorization: bearer eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTI5MSIsImVtYWlsIjoiZW1haWwtMTk0QHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NTd9.p3cHAMwwVASl1RWxrQuOMLYRZRe4rTbaIH1RRux3Q5Y"
     
Response
^^^^^^^^

Status
''''''

    200 OK

Body
''''

    {

    "app\_id": "8d0cee08-23ad-4813-a779-ad8118ea0b91",

    "guid": "c29be336-7851-4d61-a0a9-084d625b4d43"

    "policy\_json": {

    "instance\_min\_count": 1,

    "instance\_max\_count": 4,

    "scaling\_rules": [

    {

    "metric\_type": "memoryused",

    "stat\_window\_secs": 300,

    "breach\_duration\_secs": 600,

    "threshold": 30,

    "operator": "<",

    "cool\_down\_secs": 300,

    "adjustment": "-1"

    },

    {

    "metric\_type": "memoryused",

    "stat\_window\_secs": 300,

    "breach\_duration\_secs": 600,

    "threshold": 90,

    "operator": ">=",

    "cool\_down\_secs": 300,

    "adjustment": "+1"

    }

    ],

    "schedules": {

    "timezone": "Asia/Shanghai",

    "recurring\_schedule": [

    {

    "start\_time": "10:00",

    "end\_time": "18:00",

    "days\_of\_week": [

    1,

    2,

    3

    ],

    "instance\_min\_count": 1,

    "instance\_max\_count": 10,

    "initial\_min\_instance\_count": 5

    },

    {

    "start\_date": "2016-06-27",

    "end\_date": "2016-07-23",

    "start\_time": "11:00",

    "end\_time": "19:30",

    "days\_of\_month": [

    5,

    15,

    25

    ],

    "instance\_min\_count": 3,

    "instance\_max\_count": 10,

    "initial\_min\_instance\_count": 5

    },

    {

    "start\_time": "10:00",

    "end\_time": "18:00",

    "days\_of\_week": [

    4,

    5,

    6

    ],

    "instance\_min\_count": 1,

    "instance\_max\_count": 10

    },

    {

    "start\_time": "11:00",

    "end\_time": "19:30",

    "days\_of\_month": [

    10,

    20,

    30

    ],

    "instance\_min\_count": 1,

    "instance\_max\_count": 10

    }

    ],

    "specific\_date": [

    {

    "start\_date\_time": "2015-06-02T10:00",

    "end\_date\_time": "2015-06-15T13:59",

    "instance\_min\_count": 1,

    "instance\_max\_count": 4,

    "initial\_min\_instance\_count": 2

    },

    {

    "start\_date\_time": "2015-01-04T20:00",

    "end\_date\_time": "2015-02-19T23:15",

    "instance\_min\_count": 2,

    "instance\_max\_count": 5,

    "initial\_min\_instance\_count": 3

    }

    ]

    }

    },

    }

Delete Policy
~~~~~~~~~~~~~

Delete /v1/apps/:guid/policy
^^^^^^^^^^^^^^^^^^^^^^^^^^^^

Request
^^^^^^^

Route
'''''

    DELETE /v1/apps/:guid/policy

Parameters
''''''''''

+--------+-------------------------------+----------------+------------+------------------+
| Name   | Description                   | Valid values   | Required   | Example values   |
+--------+-------------------------------+----------------+------------+------------------+
| guid   | The GUID of the application   |                | true       |                  |
+--------+-------------------------------+----------------+------------+------------------+

Headers
'''''''
    Authorization: bearer
    eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTQwOCIsImVtYWlsIjoiZW1haWwtMzAzQHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NzF9.RMJZvSzCSxpj4jjZBmzbO7eoSfTAcIWVSHqFu5\_Iu\_o

cURL
''''
    | curl
      "https://[the-api-server-url]:[port]/v1/apps/8d0cee08-23ad-4813-a779-ad8118ea0b91/policy" \\
    | -X DELETE \\
    | -H "Authorization: bearer
      eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTI5MSIsImVtYWlsIjoiZW1haWwtMTk0QHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NTd9.p3cHAMwwVASl1RWxrQuOMLYRZRe4rTbaIH1RRux3Q5Y"

Response
^^^^^^^^

Status
''''''

    200 OK

Get Policy
~~~~~~~~~~

GET /v1/apps/:guid/policy
^^^^^^^^^^^^^^^^^^^^^^^^^

Request
^^^^^^^

Route
'''''

    GET /v1/apps/:guid/policy

Parameters
''''''''''

+--------+-------------------------------+----------------+------------+------------------+
| Name   | Description                   | Valid values   | Required   | Example values   |
+--------+-------------------------------+----------------+------------+------------------+
| guid   | The GUID of the application   |                | true       |                  |
+--------+-------------------------------+----------------+------------+------------------+

Headers
'''''''
    Authorization: bearer
    eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTQwOCIsImVtYWlsIjoiZW1haWwtMzAzQHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NzF9.RMJZvSzCSxpj4jjZBmzbO7eoSfTAcIWVSHqFu5\_Iu\_o

cURL
''''
    | curl
      "https://[the-api-server-url]:[port]/v1/apps/8d0cee08-23ad-4813-a779-ad8118ea0b91/policy" \\
    | -X GET \\
    | -H "Authorization: bearer
      eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJ1c2VyX2lkIjoidWFhLWlkLTI5MSIsImVtYWlsIjoiZW1haWwtMTk0QHNvbWVkb21haW4uY29tIiwic2NvcGUiOlsiY2xvdWRfY29udHJvbGxlci5hZG1pbiJdLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciJdLCJleHAiOjE0NDU1NTc5NTd9.p3cHAMwwVASl1RWxrQuOMLYRZRe4rTbaIH1RRux3Q5Y"

Response
^^^^^^^^

Status
''''''

    200 OK

Body
''''

{

"instance\_min\_count": 1,

"instance\_max\_count": 4,

"scaling\_rules": [

{

"metric\_type": "memoryused",

"stat\_window\_secs": 300,

"breach\_duration\_secs": 600,

"threshold": 30,

"operator": "<",

"cool\_down\_secs": 300,

"adjustment": "-1"

},

{

"metric\_type": "memoryused",

"stat\_window\_secs": 300,

"breach\_duration\_secs": 600,

"threshold": 90,

"operator": ">=",

"cool\_down\_secs": 300,

"adjustment": "+1"

}

],

"schedules": {

"timezone": "Asia/Shanghai",

"recurring\_schedule": [

{

"start\_time": "10:00",

"end\_time": "18:00",

"days\_of\_week": [

1,

2,

3

],

"instance\_min\_count": 1,

"instance\_max\_count": 10,

"initial\_min\_instance\_count": 5

},

{

"start\_date": "2016-06-27",

"end\_date": "2016-07-23",

"start\_time": "11:00",

"end\_time": "19:30",

"days\_of\_month": [

5,

15,

25

],

"instance\_min\_count": 3,

"instance\_max\_count": 10,

"initial\_min\_instance\_count": 5

},

{

"start\_time": "10:00",

"end\_time": "18:00",

"days\_of\_week": [

4,

5,

6

],

"instance\_min\_count": 1,

"instance\_max\_count": 10

},

{

"start\_time": "11:00",

"end\_time": "19:30",

"days\_of\_month": [

10,

20,

30

],

"instance\_min\_count": 1,

"instance\_max\_count": 10

}

],

"specific\_date": [

{

"start\_date\_time": "2015-06-02T10:00",

"end\_date\_time": "2015-06-15T13:59",

"instance\_min\_count": 1,

"instance\_max\_count": 4,

"initial\_min\_instance\_count": 2

},

{

"start\_date\_time": "2015-01-04T20:00",

"end\_date\_time": "2015-02-19T23:15",

"instance\_min\_count": 2,

"instance\_max\_count": 5,

"initial\_min\_instance\_count": 3

}

]

}

}
 
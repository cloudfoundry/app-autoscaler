==================================================
Use AutoScaler REST APIs
==================================================

.. contents:: Table of Contents

Overview
========
The Auto-Scaler API server provides a set of REST APIs to manage scaling policies, retrieve the scaling history and metric data.

Before using these APIs, your application needs to bind the Auto-Scaler service first. The following section gives the details of how to use the APIs.

Authentication
===================
For each REST API invocation, a AccessToken obtained through CloudFoundry UAA must be provided in the Authorization header, otherwise a 401 Unauthorized response will be returned. AccessToken is a credential used to access protected resources and represent an authorization issued to the client, see `User Account and Authentication - A note on Tokens`_ 
for an introdution on AccessToken and Cloud Foundry UAA.

.. _`User Account and Authentication - A note on Tokens`: https://github.com/cloudfoundry/uaa/blob/master/docs/UAA-Tokens.md 

There are two ways to get the AccessToken once you logged into Cloud Foundry using command line tool ``cf``:

* Use ``cf oauth-token`` command:

        >cf oauth-token
        
        Getting OAuth token...
        
        OK
        
        
        bearer eyJhbGciOiJIUzI1NiJ9.eyJqdGkiOiJhNjIzMGE1YS1mNzE3LTQ0YjItOWM3Yi1kNGJkYThhZGU0NjkiLCJzdWIiOiI4OGViNjM2My1hMjkzLTRlZTItYWQ1MS0yOGVkMTZmZjMwNzQiLCJzY29wZSI6WyJjbG91ZF9jb250cm9sbGVyLnJlYWQiLCJwYXNzd29yZC53cml0ZSIsImNsb3VkX2NvbnRyb2xsZXIud3JpdGUiLCJvcGVuaWQiXSwiY2xpZW50X2lkIjoiY2YiLCJjaWQiOiJjZiIsImF6cCI6ImNmIiwiZ3JhbnRfdHlwZSI6InBhc3N3b3JkIiwidXNlcl9pZCI6Ijg4ZWI2MzYzLWEyOTMtNGVlMi1hZDUxLTI4ZWQxNmZmMzA3NCIsIm9yaWdpbiI6InVhYSIsInVzZXJfbmFtZSI6InRlc3RAY24uaWJtLmNvbSIsImVtYWlsIjoidGVzdEBjbi5pYm0uY29tIiwicmV2X3NpZyI6ImQ4ZTRmNDAyIiwiaWF0IjoxNDU1NjA3NTQ1LCJleHAiOjE0NTU2NTA3NDUsImlzcyI6Imh0dHBzOi8vdWFhLmJvc2gtbGl0ZS5jb20vb2F1dGgvdG9rZW4iLCJ6aWQiOiJ1YWEiLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciIsInBhc3N3b3JkIiwiY2YiLCJvcGVuaWQiXX0.DAshF9W8w1FQadlegXVWVc0gkTWen1gXvruHwfoAepg
        
        >

* Get  ``AccessToken`` from the configuration file of ``cf`` command line tool. After you logged into Cloud Foundry, a ``.cf`` folder is created in your home directory, where you can find a JSON file named ``config.json`` that contains an entry ``AccessToken`` in the file. Please note you may need to run ``cf oauth-token`` or log into Cloud Foundry again since the ``AccessToken`` may be already expired.

        >cat ~/.cf/config.json
        
        {
        
        "ConfigVersion": 3,
        
        "Target": "https://api.bosh-lite.com",
        
        ...
        
        "AccessToken": "bearer eyJhbGciOiJIUzI1NiJ9.eyJqdGkiOiJhNjIzMGE1YS1mNzE3LTQ0YjItOWM3Yi1kNGJkYThhZGU0NjkiLCJzdWIiOiI4OGViNjM2My1hMjkzLTRlZTItYWQ1MS0yOGVkMTZmZjMwNzQiLCJzY29wZSI6WyJjbG91ZF9jb250cm9sbGVyLnJlYWQiLCJwYXNzd29yZC53cml0ZSIsImNsb3VkX2NvbnRyb2xsZXIud3JpdGUiLCJvcGVuaWQiXSwiY2xpZW50X2lkIjoiY2YiLCJjaWQiOiJjZiIsImF6cCI6ImNmIiwiZ3JhbnRfdHlwZSI6InBhc3N3b3JkIiwidXNlcl9pZCI6Ijg4ZWI2MzYzLWEyOTMtNGVlMi1hZDUxLTI4ZWQxNmZmMzA3NCIsIm9yaWdpbiI6InVhYSIsInVzZXJfbmFtZSI6InRlc3RAY24uaWJtLmNvbSIsImVtYWlsIjoidGVzdEBjbi5pYm0uY29tIiwicmV2X3NpZyI6ImQ4ZTRmNDAyIiwiaWF0IjoxNDU1NjA3NTQ1LCJleHAiOjE0NTU2NTA3NDUsImlzcyI6Imh0dHBzOi8vdWFhLmJvc2gtbGl0ZS5jb20vb2F1dGgvdG9rZW4iLCJ6aWQiOiJ1YWEiLCJhdWQiOlsiY2xvdWRfY29udHJvbGxlciIsInBhc3N3b3JkIiwiY2YiLCJvcGVuaWQiXX0.DAshF9W8w1FQadlegXVWVc0gkTWen1gXvruHwfoAepg",
        
        ...
        
        }

API EndPoint
===================

* You can obtain the URL of AutoScaler API server by checking the ``VCAP_SERVICE`` environment variable after you bind the Auto-Scaling service to your application. The ``api_url`` in the ``credentials`` part of Auto-scaling service section in the ``VCAP_SERVICE`` is the API endpoint that your application can use. Check `Binding Credentials`_ for more information about the structure and fields of service credentials in Cloud Foundry:
.. _`Binding Credentials`: http://docs.cloudfoundry.org/services/binding-credentials.html

        {
        
        "CF-AutoScaler": [
        
         {
        
          ...
        
           "name": "CF-AutoScaler-iw",
        
           "plan": "free",
        
           "credentials": {
        
             ...
        
             "api_url": "https://AutoScalingAPI.bosh-lite.com",
             
             "app_id": "aa8d19b6-eceb-4b6e-b034-926a87e98a51",
        
             "url": "https://AutoScaling.bosh-lite.com",
        
            ...
           
            }
          
          ]
          
        }
 

* The API endpoint can also be found through the ``cf env <APPNAME>`` command, it's more useful when you want to use a script file to manage Auto-Scaler service through REST API:

        >cf env <APPNAME>
        
        Getting env variables for app <APPNAME> in org <ORG> / space <SPACE> as <USERNAME>...
        
        System-Provided:
        
        {
        
          "VCAP_SERVICES": {...
        
           "CF-AutoScaler": [
           
           {
        
             "name": "CF-AutoScaler-iw",
        
             "plan": "free",
        
             "credentials": {
        
              ...
        
               "api_url": "https://AutoScalingAPI.bosh-lite.com",
               
               "app_id": "aa8d19b6-eceb-4b6e-b034-926a87e98a51",
        
               "url": "https://AutoScaling.bosh-lite.com",
        
               ...
           
              }
            
            }
            
           ]
          
        }


AppId
===================
The GUID of your application is needed when invoke the REST APIs .You can get the ``app_id`` from the ``VCAP_SERVICES`` enviroment variable, or just run the ``cf app <APPNAME> --guid`` command:

        >cf app <APPNAME> --guid
        
        aa8d19b6-eceb-4b6e-b034-926a87e98a51
        
        >
        

API Specifications 
============================================
See `AutoScaler REST API Specification`_ for detailed description of each API that Auto-Scaler provides. 

.. _`AutoScaler REST API Specification`: API_spec.rst 


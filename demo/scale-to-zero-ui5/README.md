# AI Code Camp: scale-to-zero demo

A static SAPUI5 welcome page served by the Cloud Foundry staticfile buildpack.
It displays **Hello, AI Code Camp!** and explains sleep / wake / serve.
The browser loads SAPUI5 and its Horizon Dark theme from `ui5.sap.com`; no Node.js
server, build step, background requests, or application-side autoscaling SDK is
needed. A plain HTML greeting remains visible if the CDN is unavailable.

Deploy into a dedicated demo space from the repository root:

```sh
cf create-space ai-code-camp-demo -o autoscaler-mta-1411
cf target -o autoscaler-mta-1411 -s ai-code-camp-demo
cf create-service autoscaler-mta-1411 autoscaler-free-plan ui5-logging-autoscaler
cf push UI5-logging \
  -f demo/scale-to-zero-ui5/manifest.yml \
  --var app_name=UI5-logging \
  --var app_host=ui5-logging \
  --var app_domain=autoscaler.app-runtime-interfaces.ci.cloudfoundry.org \
  -c null --no-start
cf bind-service UI5-logging ui5-logging-autoscaler \
  -c demo/scale-to-zero-ui5/policy.json
cf start UI5-logging
```

`-c null` clears an existing Go/binary start-command override so the staticfile
buildpack can run NGINX. An existing autoscaler service binding and policy are
preserved by the push. The included v0.1 policy allows zero to one instances,
scaling in when CPU remains below 40% for 60 seconds. The autoscaler offering
must be enabled for the demo org.

Open the application's **root URL** to see the UI5 page; `/health` is a separate
plain-text health endpoint. Let the app reach zero instances before the demo.
The first navigation wakes it. The **Say hello again** button navigates with a
fresh query parameter so browser caching does not hide a later wake-up.

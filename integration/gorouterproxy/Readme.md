# Why Gorouter Proxy?

Gorouter Proxy is a component that sits between the client and the downstream services.
It acts like a real CF gorouter which adds x-forwarded-client-cert (XFCC) and forwards the request to the downstream service based on the requested port.

Gorouter Proxy also terminates the TLS and forwards only http traffic to the downstream service. With the move of autoscaler bosh-deployed services to cf based application, this is required as the downstream services accepts http connection (not https) only






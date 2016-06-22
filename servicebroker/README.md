autoscaler-service-broker
=====

AutoScaler service broker is built by nodejs/expressjs

---
**Local dependencies**
```sh
npm run bootstrap
```
**Service broker configuration**

Change configuration for service broker in file `config/settings.json`, including the following items:
* `port`, the port number in which the service broker app is launched and listening to.  The default value is 8080.
* `user` and `password` : HTTP basic authorization is enabled according to the [CF service broker API # authorization](http://docs.cloudfoundry.org/services/api.html#authentication).  Please specific the user/password in setting.json, and use the same user/password when issue command "cf create-service-broker"


**Run eslint**
```js
npm run lint
```
**Run unit test**
```js
npm run test
```

**Start your server using nodemon**
```js
npm run start
```
Access with `http://localhost:8080/v2/catalog`

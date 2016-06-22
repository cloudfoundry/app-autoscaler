autoscaler-service-broker
=====

AutoScaler service broker is built by nodejs/expressjs

---
**Service broker configuration**

Change configuration for service broker in file `config/settings.json`, including the following items:
* `port`, the port number in which the service broker app is launched and listening to.  The default value is 8080. 
* `user` and `password` : HTTP basic authorization is enabled according to the [CF service broker API # authorization](http://docs.cloudfoundry.org/services/api.html#authentication).  Please specific the user/password in setting.json, and use the same user/password when issue command "cf create-service-broker"
* `dbServer` : the connection url for the database server. The default value is using postgres in local with a no-password user "postgres"
* `dbName` : the database name for service broker. The default value is "servicebroker_db"

**Prerequisite**
* install [postgres](https://www.postgresql.org/download/) according to the `dbServer` definition in `config/settings.json`
* create a no-password superuser for postgres with command:
```sh
createuser postgres -s
```

**Local dependencies**
```sh
npm bootstrap
```

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

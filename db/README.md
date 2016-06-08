## Build the db app
Provide the DB credentials and url in the /src/main/resources/application.properties file.
```
mvn clean package
```

## Deploy
Requires java buildpack version 3.7
```
cf push autoscaler-db -p <PATH_TO_REPO>/db/target/db-1.0-SNAPSHOT.war -b https://github.com/cloudfoundry/java-buildpack.git#v3.7
```
# Autoscaler - Scheduler

## Database

### Create tables

#### Create Scheduler tables
```sh
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=scheduler/db/scheduler.changelog-master.yaml --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
```

#### Create Quartz Scheduler tables
```sh
java -cp 'db/target/lib/*'  liquibase.integration.commandline.Main --changeLogFile=scheduler/db/quartz.changelog-master.yaml --url jdbc:postgresql://127.0.0.1/autoscaler --driver=org.postgresql.Driver update
```

## Package 

### Skip Unit Test

```sh
mvn clean package -Dmaven.test.skip=true
```

### Package and run Unit Test

```sh
mvn clean package
```

## Run Unit Tests 


### All

```sh
mvn test
```

### To run specific class tests specify the test class name

#### For example to run ScheduleManagerTest
```sh
mvn -Dtest=ScheduleManagerTest test
```

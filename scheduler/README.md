# Autoscaler - Scheduler

## Package 

### Skip Unit Test

```sh
mvn clean package -Dmaven.test.skip=true
```

### Package and run Unit Test)

```sh
mvn clean package
```

## Run Unit Tests 

Note: The unit test are using in memory database.

### All

```sh
mvn test
```

### Dao

```sh
mvn -Dtest=ScheduleDaoImplTest test
```

### Service

```sh
mvn -Dtest=ScheduleManagerTest test
mvn -Dtest=ScalingJobManagerTest test
```
### Rest
mvn -Dtest=ScheduleRestApiTest test



URL := jdbc:mysql://127.0.0.1/autoscaler
DRIVER := com.mysql.cj.jdbc.Driver
DB_USER = root

target/lib:
	mvn package -DskipTests

createdb: target/lib
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../src/autoscaler/api/db/api.db.changelog.yml --username=$(DB_USER) update
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../src/autoscaler/servicebroker/db/servicebroker.db.changelog.json --username=$(DB_USER) update
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../scheduler/db/scheduler.changelog-master.yaml --username=$(DB_USER) update
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../scheduler/db/quartz.changelog-master.yaml --username=$(DB_USER) update
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../src/autoscaler/metricsserver/db/metricscollector.db.changelog.yml --username=$(DB_USER) update
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../src/autoscaler/eventgenerator/db/dataaggregator.db.changelog.yml --username=$(DB_USER) update
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../src/autoscaler/scalingengine/db/scalingengine.db.changelog.yml --username=$(DB_USER) update
	java -cp 'target/lib/*' liquibase.integration.commandline.Main --url $(URL) --driver=$(DRIVER) --changeLogFile=../src/autoscaler/operator/db/operator.db.changelog.yml --username=$(DB_USER) update

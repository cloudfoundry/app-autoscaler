#!/bin/bash

set -e
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT=$(readlink -f "$SCRIPT_DIR/..")
DB_USER=${DB_USER:-"root"}
java -cp "$ROOT/src/autoscaler/api/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="api.db.changelog.yml" --username="${DB_USER}" update
java -cp "$ROOT/src/autoscaler/servicebroker/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="$ROOT/src/autoscaler/servicebroker/db/servicebroker.db.changelog.json" --username="${DB_USER}" update
java -cp "$ROOT/scheduler/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="$ROOT/scheduler/db/scheduler.changelog-master.yaml" --username="${DB_USER}" update
java -cp "$ROOT/scheduler/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="$ROOT/scheduler/db/quartz.changelog-master.yaml" --username="${DB_USER}" update
java -cp "$ROOT/src/autoscaler/metricsserver/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="$ROOT/src/autoscaler/metricsserver/db/metricscollector.db.changelog.yml" --username="${DB_USER}" update
java -cp "$ROOT/src/autoscaler/eventgenerator/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="dataaggregator.db.changelog.yml" --username="${DB_USER}" update
java -cp "$ROOT/src/autoscaler/scalingengine/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="scalingengine.db.changelog.yml" --username="${DB_USER}" update
java -cp "$ROOT/src/autoscaler/operator/db/:$ROOT/db/target/lib/*" liquibase.integration.commandline.Main --url jdbc:mysql://127.0.0.1/autoscaler --driver=com.mysql.cj.jdbc.Driver --changeLogFile="operator.db.changelog.yml" --username="${DB_USER}" update
#!/bin/bash

set -e
LOG_FILE=liqubase.log
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"
ROOT="$SCRIPT_DIR/.."
DB_USER=${DB_USER:-"root"}

files="api.db.changelog.yml \
       servicebroker.db.changelog.json \
       scheduler.changelog-master.yaml \
       quartz.changelog-master.yaml \
       metricscollector.db.changelog.yml \
       dataaggregator.db.changelog.yml \
       scalingengine.db.changelog.yml \
       operator.db.changelog.yml"

class_path="$ROOT/src/autoscaler/api/db/:\
$ROOT/db/target/lib/*:\
$ROOT/src/autoscaler/servicebroker/db/:\
$ROOT/scheduler/db/:\
$ROOT/scheduler/db/:\
$ROOT/src/autoscaler/metricsserver/db/:\
$ROOT/src/autoscaler/eventgenerator/db/:\
$ROOT/src/autoscaler/scalingengine/db/:\
$ROOT/src/autoscaler/operator/db/"

[ -e "${LOG_FILE}" ] && rm "${LOG_FILE}"
trap 'error' ERR
trap "rm ${LOG_FILE} || echo \"no log file\"" EXIT

error() {
  [ -e "${LOG_FILE}" ] && cat "${LOG_FILE}"
}
echo "# Applying liquibase change sets to: 'mysql://127.0.0.1/autoscaler'"
for file in $files; do
  echo "  - applying: '$file'" | tee -a "${LOG_FILE}"
  java -cp "$class_path" liquibase.integration.commandline.Main \
    --url jdbc:mysql://127.0.0.1/autoscaler\
    --driver=com.mysql.cj.jdbc.Driver\
    --changeLogFile="$file"\
    --username="${DB_USER}"\
     update >> "${LOG_FILE}"
done

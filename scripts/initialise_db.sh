#!/usr/bin/env bash

set -euo pipefail

echo "Running $0"

DB_HOST="${DB_HOST:-localhost}"
LOG_FILE="liquibase.log"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
ROOT="$SCRIPT_DIR/.."

usage() {
  echo -e "Usage: $0 [postgres|mysql]" >&2
  echo "  Error: Please provide either postgres or mysql as a parameter" >&2
  exit 1
}

if [ "$#" -ne 1 ]; then
  usage
fi

case "$1" in
  mysql)
    DB_USER="${DB_USER:-root}"
    URL="jdbc:mysql://${DB_HOST}/autoscaler"
    DRIVER="com.mysql.cj.jdbc.Driver"
    PASSWORD_OPT=""
    ;;
  postgres)
    DB_USER="${DB_USER:-postgres}"
    DB_PASSWORD="${DB_PASSWORD:-postgres}"
    URL="jdbc:postgresql://${DB_HOST}/autoscaler"
    DRIVER="org.postgresql.Driver"
    PASSWORD_OPT="--password=${DB_PASSWORD}"
    ;;
  *)
    usage
    ;;
esac

files="api.db.changelog.yml
       servicebroker.db.changelog.yaml
       scheduler.changelog-master.yaml
       quartz.changelog-master.yaml
       metricscollector.db.changelog.yml
       dataaggregator.db.changelog.yml
       scalingengine.db.changelog.yml
       operator.db.changelog.yml"

class_path="$ROOT/dbtasks/target/lib/*:\
$ROOT/scheduler/db/:\
$ROOT/api/db/:\
$ROOT/servicebroker/db/:\
$ROOT/metricsserver/db/:\
$ROOT/eventgenerator/db/:\
$ROOT/scalingengine/db/:\
$ROOT/operator/db/"

[ -e "${LOG_FILE}" ] && rm "${LOG_FILE}"

error() {
  [ -e "${LOG_FILE}" ] && cat "${LOG_FILE}"
}
trap error ERR
trap 'rm -f "${LOG_FILE}" || echo "no log file"' EXIT

echo "# Applying liquibase change sets to: '${URL}'"
for file in ${files}; do
  echo "  - applying: '$file'" | tee -a "${LOG_FILE}"
  # shellcheck disable=SC2086
  java -cp "${class_path}" liquibase.integration.commandline.Main \
    --url="${URL}" \
    --driver="${DRIVER}" \
    --changeLogFile="${file}" \
    --username="${DB_USER}" \
    ${PASSWORD_OPT} \
    update >> "${LOG_FILE}"
done


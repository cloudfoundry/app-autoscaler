#!/bin/bash
# shellcheck disable=SC2155

set -euo pipefail

JAVA_BIN=${JAVA_BIN:-"/home/vcap/app/.java-buildpack/open_jdk_jre/bin/java"}
CERTS_DIR="$(mktemp -d)"
mkdir -p "$CERTS_DIR"

if [ ! -f "$JAVA_BIN" ]; then
	echo "Java binary not found at $JAVA_BIN"
	exit 1
fi

extract_service() {
  local json="$1"
  local tag="$2"
  echo "$json" | jq -r --arg tag "$tag" '.["postgresql-db"][]?, .["user-provided"][]? | select(.tags[] == $tag) | .credentials'
}

parse_uri() {
  local uri="$1"
  local field="$2"
  case "$field" in
    user) echo "$uri" | awk -F[:@] '{print substr($2, 3)}' ;;
    password) echo "$uri" | awk -F[:@] '{print $3}' ;;
    host) echo "$uri" | awk -F[@:/?] '{print $6}' ;;
    port) echo "$uri" | awk -F[@:/?] '{print $7}' ;;
    dbname) echo "$uri" | awk -F[@:/?] '{print $8}' ;;
  esac
}

persist_cert() {
  local content="$1"
  local file="$2"
  echo "Persisting cert to $file"
  if [ "$content" == "null" ]; then
    return
  fi

  echo "$content" > "$file"
  chmod 600 "$file"
}


build_jdbc_url() {
  local host="$1" port="$2" dbname="$3" client_cert="$4" client_key="$5" server_ca="$6"
  local url_params=""
  local client_pk8_key="$CERTS_DIR/client-key.pk8"

  if [ -s "$client_cert" ]; then
    url_params="&sslcert=$client_cert"
  fi

  if [ -s "$server_ca" ]; then
    url_params="$url_params&sslrootcert=$server_ca"
  fi

  if [ -s "$client_key" ]; then
    convert_to_pk8 "$client_key" "$client_pk8_key"
    url_params="$url_params&sslkey=$client_pk8_key&sslmode=verify-ca"
  fi

  echo "jdbc:postgresql://$host:$port/$dbname?$url_params"

}

function convert_to_pk8() {
  local -r in_file="$1"
  local -r out_file="$2"
  openssl pkcs8 -topk8 -outform DER -in "$in_file" -out "$out_file" -nocrypt
  chgrp vcap "$out_file"
  chmod g+r "$out_file"
}


function run_liquibase() {
  local -r jdbcdburl="$1"
  local -r user="$2"
  local -r password="$3"
  local -r changelog="$4"

  local classpath=$(readlink -f /home/vcap/app/BOOT-INF/lib/* | tr '\n' ':')

  "$JAVA_BIN" -cp "$classpath" liquibase.integration.commandline.Main \
    --url "$jdbcdburl" --username="$user" --password="$password" --driver=org.postgresql.Driver --logLevel=DEBUG --changeLogFile="$changelog" update
}

function main() {
  local changelogs=("$@")
  local json="$VCAP_SERVICES"
  local tag="policy_db"

  local service=$(extract_service "$json" "$tag")
  local uri=$(echo "$service" | jq -r '.uri')

  local host=$(parse_uri "$uri" "host")
  local port=$(parse_uri "$uri" "port")
  local dbname=$(parse_uri "$uri" "dbname")

  local client_cert="$CERTS_DIR/client-cert.pem"
  local client_key="$CERTS_DIR/client-key.pem"
  local server_ca="$CERTS_DIR/server-ca.pem"


  persist_cert "$(echo "$service" | jq -r '.client_cert // .sslcert')" "$client_cert"
  persist_cert "$(echo "$service" | jq -r '.server_ca // .sslrootcert')" "$server_ca"
  persist_cert "$(echo "$service" | jq -r '.client_key // .sslkey')" "$client_key"

  JDBCDBURL=$(build_jdbc_url "$host" "$port" "$dbname" "$client_cert" "$client_key" "$server_ca")

  PASSWORD=$(parse_uri "$uri" "password")
  USER=$(parse_uri "$uri" "user")




  for changelog in "${changelogs[@]}"; do
    run_liquibase "$JDBCDBURL" "$USER" "$PASSWORD" "$changelog"
  done
}


main "$@"


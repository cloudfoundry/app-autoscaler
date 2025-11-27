#!/usr/bin/env bash
set -eu
export PATH=${HOME}/go/bin:${PATH}
###################################################################################################
 # This script downloads google formatter and displays formatting issues on Github actions
###################################################################################################
GOOGLE_JAR_VERSION=${GOOGLE_JAR_VERSION:-"1.22.0"}
GOOGLE_JAR_NAME=${GOOGLE_JAR_NAME:-"google-java-format-${GOOGLE_JAR_VERSION}-all-deps.jar"}
! [[ -e "$GOOGLE_JAR_NAME" ]] && \
  curl -fLJO "https://github.com/google/google-java-format/releases/download/v$GOOGLE_JAR_VERSION/$GOOGLE_JAR_NAME"
# shellcheck disable=SC2046
files_to_be_formatted=$(java \
              -jar "${GOOGLE_JAR_NAME}" --dry-run --skip-javadoc-formatting $(find scheduler -name '*.java' ! -name 'CloudFoundryConfigurationProcessorTest.java'))

if  [[ -n "$files_to_be_formatted" ]]; then
  echo "Formatter Results..."
  echo "Files require reformatting:"
  echo "${files_to_be_formatted}"
  exit 1
fi

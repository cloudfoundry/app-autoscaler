#!/usr/bin/env bash

###################################################################################################
 # This pre-commit hook displays Java formatting issues
###################################################################################################
set -e -o pipefail

REPO_PATH=$(git rev-parse --show-toplevel)
DOWNLOAD_PATH="$REPO_PATH"/.cache
echo "$DOWNLOAD_PATH"
function download_google_formatter {
    cd "$DOWNLOAD_PATH"
    if [ ! -f "$DOWNLOAD_PATH"/google-java-format-1.10.0-all-deps.jar ] ; then
        curl -LJO "https://github.com/google/google-java-format/releases/download/v1.10.0/google-java-format-1.10.0-all-deps.jar"
        chmod 755 google-java-format-1.10.0-all-deps.jar
    fi
}
changed_java_files=$(git diff --cached --name-only --diff-filter=ACMR | grep ".*java$" )
if  [[ -n "$changed_java_files" ]]; then
    echo "$DOWNLOAD_PATH"
    mkdir -p "$DOWNLOAD_PATH"
    download_google_formatter
    cd "$REPO_PATH"
fi

# dry run Google formatter
echo "download path : $DOWNLOAD_PATH"/google-java-format-1.10.0-all-deps.jar
files_to_be_changed=$( java -jar "$DOWNLOAD_PATH"/google-java-format-1.10.0-all-deps.jar -n $changed_java_files)

# if there are any stage changes
if  [[ -n "$files_to_be_changed" ]]; then
    echo "Formatter Results:"
    echo "Analyzing java file(s) using google-java-format-1.10.0-all-deps.jar..."
    echo "Incorrect formatting found:"


    files_to_be_changed=$(echo $files_to_be_changed|tr -d '\n')
    echo "$files_to_be_changed"
    echo "Please correct the formatting of the files(s) using one of the following options:"
    echo "  Reformat Code - IntelliJ or Format Document - Eclipse"
    echo "  java -jar "$DOWNLOAD_PATH"/google-java-format-1.10.0-all-deps.jar -replace ${files_to_be_changed[*]}"


    exit 2
fi
  java -jar /Users/I545443/sap/asalan316/app-autoscaler/.cache/google-java-format-1.10.0-all-deps.jar -replace  scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/SchedulerApplication.java  scheduler/src/main/java/org/cloudfoundry/autoscaler/scheduler/rest/ControllerExceptionHandler.java

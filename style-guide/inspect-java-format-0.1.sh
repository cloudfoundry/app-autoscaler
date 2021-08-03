#!/usr/bin/env bash

###################################################################################################
 # This pre-commit hook displays Java formatting issues
###################################################################################################
set -e -o pipefail

REPO_PATH=$(git rev-parse --show-toplevel)
CHECKSTYLE_CONFIG="google_checks.xml"
CHECKSTYLE_CONFIG_PATH="$REPO_PATH"/style-guide/"$CHECKSTYLE_CONFIG"
DOWNLOAD_PATH="$REPO_PATH"/.cache
CHECKSTYLE_JAR_NAME="checkstyle-8.44-all.jar"
JAVA_SOURCES_PATH="$REPO_PATH"/scheduler/src/
echo "Current Configs..."
echo "REPO_PATH: $REPO_PATH"
echo "DOWNLOAD_PATH: $DOWNLOAD_PATH"
echo "JAVA_SOURCES_PATH: $JAVA_SOURCES_PATH"
echo "CHECKSTYLE_JAR_NAME: $CHECKSTYLE_JAR_NAME"
echo "CHECKSTYLE_CONFIG: $CHECKSTYLE_CONFIG"
echo "CHECKSTYLE_CONFIG_PATH: $CHECKSTYLE_CONFIG_PATH"

function download_google_formatter {
    cd "$DOWNLOAD_PATH"
    if [ ! -f "$DOWNLOAD_PATH"/google-java-format-1.10.0-all-deps.jar ] ; then
        curl -LJO "https://github.com/google/google-java-format/releases/download/v1.10.0/google-java-format-1.10.0-all-deps.jar"
        chmod 755 google-java-format-1.10.0-all-deps.jar
    fi
}

function download_checkstyle {
    cd "$DOWNLOAD_PATH"
    echo "I am download_checkstyle"
    if [ ! -f "$DOWNLOAD_PATH"/"$CHECKSTYLE_JAR_NAME" ] ; then
        curl -LJO "https://github.com/checkstyle/checkstyle/releases/download/checkstyle-8.44/checkstyle-8.44-all.jar"
        chmod 755 "$CHECKSTYLE_JAR_NAME"
    fi
}



if  [[ -n "$changed_java_files" ]]; then
    mkdir -p "$DOWNLOAD_PATH"
    download_google_formatter
    download_checkstyle
    cd "$REPO_PATH"

fi

download_checkstyle
changed_java_files=$(git diff --cached --name-only --diff-filter=ACMR | grep ".*java$" )
echo changed_java_files
echo "...."

    #checkstyle
    echo "executing java -jar "$DOWNLOAD_PATH"/"$CHECKSTYLE_JAR_NAME" -c "$CHECKSTYLE_CONFIG_PATH" $changed_java_files"
    checkstyle_result=$(java -jar "$DOWNLOAD_PATH"/"$CHECKSTYLE_JAR_NAME" -c "$CHECKSTYLE_CONFIG_PATH" $JAVA_SOURCES_PATH )
    echo "exiting"
    echo "results: $checkstyle_result"

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
    echo "  - java -jar "$DOWNLOAD_PATH"/google-java-format-1.10.0-all-deps.jar -replace ${files_to_be_changed[*]}"
    echo "  - Reformat Code - IntelliJ or Format Document - Eclipse (google code style required)"
    exit 2
fi

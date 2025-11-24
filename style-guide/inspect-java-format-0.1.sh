#!/usr/bin/env bash

###################################################################################################
 # This pre-commit hook displays Java formatting issues
###################################################################################################
set -e -o pipefail

REPO_PATH=$(git rev-parse --show-toplevel)
DOWNLOAD_PATH="$REPO_PATH"/.cache
CHECKSTYLE_JAR_NAME="checkstyle-8.44-all.jar"
GOOGLE_JAR_VERSION="1.11.0"
GOOGLE_JAR_NAME="google-java-format-${GOOGLE_JAR_VERSION}-all-deps.jar"
script_dir="$( cd -- "$(dirname "$0")" >/dev/null 2>&1 ; pwd -P )"
result_log="${script_dir}/result.txt"
CHECKSTYLE_CONFIG_FILE_NAME="google_checks.xml"
CHECKSTYLE_CONFIG_PATH="$REPO_PATH"/style-guide

function download_google_formatter {
    cd "$DOWNLOAD_PATH"
    if [ ! -f "$DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME" ] ; then
        curl -LJO "https://github.com/google/google-java-format/releases/download/v$GOOGLE_JAR_VERSION/$GOOGLE_JAR_NAME" -o "$GOOGLE_JAR_NAME"
        chmod 755 "$GOOGLE_JAR_NAME"
    fi
}

function download_checkstyle {
    cd "$DOWNLOAD_PATH"
    if [ ! -f "$DOWNLOAD_PATH"/"$CHECKSTYLE_JAR_NAME" ] ; then
        curl -LJO "https://github.com/checkstyle/checkstyle/releases/download/checkstyle-8.44/checkstyle-8.44-all.jar"
        chmod 755 "$CHECKSTYLE_JAR_NAME"
    fi
}

# grep only stage java files
changed_java_files=$(git diff --cached --name-only --diff-filter=ACMR \
      | grep ".*java$" \
      | tr '\n' ' ')

if  [[ -n "$changed_java_files" ]]; then
    mkdir -p "$DOWNLOAD_PATH"
    download_google_formatter
    download_checkstyle
    cd "$REPO_PATH"
fi

#####################################################################
#   Checkstyle                                                      #
#####################################################################
echo "Running Checkstyle using $DOWNLOAD_PATH"/"$CHECKSTYLE_JAR_NAME..."
trap "rm ${result_log} || echo " EXIT
CHECKSTYLE_OUTPUT=$(java \
  --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED \
  -jar "$DOWNLOAD_PATH"/"$CHECKSTYLE_JAR_NAME" \
  -p "${CHECKSTYLE_CONFIG_PATH}"/checkstyle.properties \
  -c "${CHECKSTYLE_CONFIG_PATH}"/"${CHECKSTYLE_CONFIG_FILE_NAME}" \
  -o "${result_log}" $changed_java_files)

FILES_NEEDS_CORRECTION=$(cat "${result_log}" | grep -v "Starting audit..." | { grep -v "Audit done" || true; } )

if  [[ -n "$FILES_NEEDS_CORRECTION" ]]; then
  echo "$FILES_NEEDS_CORRECTION"
  exit 1
fi

#####################################################################
#   Google Formatter                                                #
#####################################################################
echo "============================================================"
echo "Google Formatting using $DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME..."
files_to_be_formatted=$( java \
                    --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED \
                    -jar "$DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME"  -n --skip-javadoc-formatting $changed_java_files)

# if there are any stage changes
if  [[ -n "$files_to_be_formatted" ]]; then
    echo "Incorrect formatting found:"
    echo -e "These file(s) have formatting issues: \n$files_to_be_formatted\n"
    files_to_be_formatted=$(echo "$files_to_be_formatted" |tr '\n' ' ')
    echo "Please correct the formatting of the files(s) using one of the following options:"
    echo "   java --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED  -jar "$DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME" -replace --skip-javadoc-formatting $files_to_be_formatted"
    exit 2
fi



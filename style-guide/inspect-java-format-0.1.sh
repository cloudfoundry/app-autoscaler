#!/usr/bin/env bash

###################################################################################################
 # This pre-commit hook displays Java formatting issues
###################################################################################################
set -e -o pipefail

REPO_PATH=$(git rev-parse --show-toplevel)
JAVA_SOURCES_PATH="$REPO_PATH"/scheduler/src/
DOWNLOAD_PATH="$REPO_PATH"/.cache
CHECKSTYLE_JAR_NAME="checkstyle-8.44-all.jar"
GOOGLE_JAR_NAME="google-java-format-1.11.0-all-deps.jar"
GOOGLE_JAR_VERSION="1.11.0"

CHECKSTYLE_CONFIG_FILE_NAME="google_checks.xml"
CHECKSTYLE_CONFIG_PATH="$REPO_PATH"/style-guide

echo "Current Configs..."
echo "REPO_PATH: $REPO_PATH"
echo "DOWNLOAD_PATH: $DOWNLOAD_PATH"
echo "JAVA_SOURCES_PATH: $JAVA_SOURCES_PATH"
echo "CHECKSTYLE_JAR_NAME: $CHECKSTYLE_JAR_NAME"
echo "CHECKSTYLE_CONFIG_FILE_NAME: CHECKSTYLE_CONFIG_FILE_NAME"
echo "CHECKSTYLE_CONFIG_PATH: $CHECKSTYLE_CONFIG_PATH"

function download_google_formatter {
    cd "$DOWNLOAD_PATH"
    if [ ! -f "$DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME" ] ; then
        curl -LJO "https://github.com/google/google-java-format/releases/download/v$GOOGLE_JAR_VERSION/$GOOGLE_JAR_NAME"
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
changed_java_files=$(git diff --cached --name-only --diff-filter=ACMR | grep ".*java$" )
if  [[ -n "$changed_java_files" ]]; then
    mkdir -p "$DOWNLOAD_PATH"
    download_google_formatter
    download_checkstyle
    cd "$REPO_PATH"
fi
# replace newlines with space
changed_java_files=$(echo "$changed_java_files" |tr '\n' ' ')

#####################################################################
#   Checkstyle                                                      #
#####################################################################
java \
  --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED \
  -jar "$DOWNLOAD_PATH"/"$CHECKSTYLE_JAR_NAME" -p "${CHECKSTYLE_CONFIG_PATH}"/checkstyle.properties -c "${CHECKSTYLE_CONFIG_PATH}"/"${CHECKSTYLE_CONFIG_FILE_NAME}" $changed_java_files


#####################################################################
#   Google Formatter                                                #
#####################################################################
echo "============================================================"
echo "Google Formatting using $DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME..."
echo java java \
  --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED \
  --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED \
  -jar "$DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME" \
  -n --skip-javadoc-formatting \
  "src\scheduler"
files_to_be_formatted=$( java \
                    --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED \
                    --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED \
                    -jar "$DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME"  -n --skip-javadoc-formatting $changed_java_files)
# if there are any stage changes
if  [[ -n "$files_to_be_formatted" ]]; then
    echo "Formatter Results:"
    echo "Analyzing java file(s) using "$GOOGLE_JAR_NAME"..."
    echo "Incorrect formatting found:"
    files_to_be_formatted=$(echo "$files_to_be_formatted" |tr '\n' ' ')
    echo "$files_to_be_formatted"
    echo "Please correct the formatting of the files(s) using one of the following options:"
    echo "   java -jar "$DOWNLOAD_PATH"/"$GOOGLE_JAR_NAME" -replace --skip-javadoc-formatting $files_to_be_formatted"
    echo "   Reformat Code - IntelliJ or Format Document - Eclipse (google code style required)"
    exit 2
fi



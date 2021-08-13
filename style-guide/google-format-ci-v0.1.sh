#!/usr/bin/env bash

###################################################################################################
 # This script downloads google formatter and displays formatting issues on Github actions
###################################################################################################

set -e -o pipefail

curl -LJO "https://github.com/google/google-java-format/releases/download/v$GOOGLE_JAR_VERSION/$GOOGLE_JAR_NAME"
files_to_be_formatted=$(java \
              --add-exports jdk.compiler/com.sun.tools.javac.api=ALL-UNNAMED \
              --add-exports jdk.compiler/com.sun.tools.javac.file=ALL-UNNAMED \
              --add-exports jdk.compiler/com.sun.tools.javac.parser=ALL-UNNAMED \
              --add-exports jdk.compiler/com.sun.tools.javac.tree=ALL-UNNAMED \
              --add-exports jdk.compiler/com.sun.tools.javac.util=ALL-UNNAMED \
              -jar "${GOOGLE_JAR_NAME}" -n --skip-javadoc-formatting "$(find . -name '*.java')")

if  [[ -n "$files_to_be_formatted" ]]; then
  echo "Formatter Results..."
  echo "Files reformatting required:"
  echo "${files_to_be_formatted}"
  exit 1
fi
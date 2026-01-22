#!/usr/bin/env bash
set -eu
readarray -t files_to_be_checked < <(find scheduler -name '*.java')
files_to_be_formatted=$(google-java-format --dry-run --skip-javadoc-formatting "${files_to_be_checked[@]}")

if  [[ -n "$files_to_be_formatted" ]]; then
  echo "Formatter Results..."
  echo "Files require reformatting:"
  echo "${files_to_be_formatted}"
  exit 1
fi

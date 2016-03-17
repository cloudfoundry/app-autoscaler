#!/bin/bash

set -e

ROOT_DIR=$(cd $(dirname $(dirname $0)) && pwd)

go build -o $ROOT_DIR/out/as_plugin

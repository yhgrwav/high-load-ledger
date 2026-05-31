#!/bin/sh
set -e

if [ "$LOAD_GEN_WORKING" != "true" ]; then
  echo "loadgen: disabled (set LOAD_GEN_WORKING=true to run)"
  exit 0
fi

exec /app/loadgen

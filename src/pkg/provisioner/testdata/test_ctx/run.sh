#!/bin/bash

if [[ ! -f run.sh ]]; then
  echo "Cannot find self in working directory (PWD is incorrect)"
  exit 1
fi

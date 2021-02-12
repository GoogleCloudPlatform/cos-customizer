#!/bin/bash

if [[ -z "${TMPDIR}" ]]; then
  echo "TMPDIR is missing in environment (parent env is not propagated)"
  exit 1
fi
if [[ -z "${TEST}" ]]; then
  echo "TEST is missing in environment"
  echo "(user provided env may not be propagated)"
  exit 1
fi

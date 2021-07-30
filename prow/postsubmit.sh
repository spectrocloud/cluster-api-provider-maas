#!/bin/bash
########################################
# Postsubmit script triggered by Prow. #
########################################

WD=$(dirname $0)
WD=$(cd $WD; pwd)
ROOT=$(dirname $WD)
source prow/functions.sh

# Exit immediately for non zero status
set -e
# Check unset variables
set -u
# Print command trace
set -x

build_code
run_lint

exit 0

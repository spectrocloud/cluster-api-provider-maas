#!/bin/bash
########################################
# Presubmit script triggered by Prow.  #
########################################
action=$1
if [[ ! ${action} ]]; then
    action='default'
fi

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
run_tests
run_lint

if [[ ${action} == "build_artifacts" ]]; then
    create_images
    create_manifest 
    delete_images
fi

if [[ ${action} == "code_coverage" ]]; then
    run_sonar_lint
fi

if [[ ${action} == "compliance_scan" ]]; then
    create_images
    run_container_scan
    delete_images
fi

exit 0

#!/bin/bash
########################################
# Daily job script triggered by Prow.  #
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

check_pre_released
build_code

create_images
create_manifest ${REPO_NAME}
create_release_manifest 

delete_images
exit 0

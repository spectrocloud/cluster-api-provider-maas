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

make_release

delete_images
exit 0

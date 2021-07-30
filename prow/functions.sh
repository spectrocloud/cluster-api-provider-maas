# Common set of functions
# Error check is done with set -e command . Build will fail if any of the commands fail

# Variables expected from CI - PULL_NUMBER , JOB_TYPE , ARTIFACTS , SONAR_SCAN_TOKEN, SONARQUBE_URL, DOCKER_REGISTRY
DATE=$(date '+%Y%m%d')

print_step() {
	text_val=$1
	set +x
	echo " "
	echo "###################################################
#  ${text_val}	
###################################################"
	echo " "
	set -x
}

set_image_tag() {
	IMG_TAG="latest"
        IMG_PATH=""
	
	if [[ ${JOB_TYPE} == 'presubmit' ]]; then
	    VERSION_SUFFIX="-dev"
	    IMG_LOC='pr'	
	    IMG_TAG=${PULL_NUMBER}
	    PROD_BUILD_ID=${IMG_TAG}	
            IMG_PATH=spectro-capi-external/${IMG_LOC}
        fi
	if [[ ${JOB_TYPE} == 'periodic' ]]; then
	    VERSION_SUFFIX="-$(date +%m%d%y)"
	    IMG_LOC='daily'	
	    IMG_TAG=$(date +%Y%m%d.%H%M)
	    PROD_BUILD_ID=${IMG_TAG}	
            IMG_PATH=spectro-capi-external/${IMG_LOC}
	fi
	if [[ ${SPECTRO_RELEASE} ]] && [[ ${SPECTRO_RELEASE} == "yes" ]]; then
	    export VERSION_SUFFIX=""
	    IMG_LOC='release'	
	    IMG_TAG=$(make version)
	    PROD_BUILD_ID=$(date +%Y%m%d.%H%M)
            IMG_PATH=spectro-images-public/${IMG_LOC}
	    OVERLAY=overlays/release
	    DOCKER_REGISTRY=${DOCKER_REGISTRY_CLIENT}
	fi

	export PROD_BUILD_ID	
	export IMG_PATH
	export IMG_TAG 
	export VERSION_SUFFIX
	export PROD_VERSION=$(make version)
}

set_release_vars() {
	RELEASE_DIR=gs://capi-prow-artifacts/release/${REPO_NAME}
	VERSION_DIR=${RELEASE_DIR}/${PROD_VERSION}
	MARKER_FILE=marker
}

build_code() {
	print_step "Building Code"
	make manager

	print_step "Copy binary to artifacts"
        if [[ -d bin ]]; then
                gsutil cp -r bin ${ARTIFACTS}/bin
        fi
}

run_tests() {
  	print_step "Running Tests"
  	make test
}

create_images() {
	print_step "Create and Push the images"
	make docker-build
	make docker-push
}

delete_images() {
	print_step "Delete local images"
	make docker-rmi
}


create_manifest() {
	project_name=${REPO_NAME}
	print_step "Create manifest files and copy to artifacts folder"
	# Manifest output has all secrets printed. Mask the output
	make manifests > /dev/null 2>&1

	mkdir -p ${ARTIFACTS}/${project_name}/build
	mkdir -p ${ARTIFACTS}/${project_name}/manifests
	cp -r config ${ARTIFACTS}/${project_name}/build/kustomize

	if [[ -d _build/manifests ]]; then
		cp -r _build/manifests ${ARTIFACTS}/${project_name} 
	fi 
}

run_lint() {
	print_step "Running Lint check"
	golangci-lint run    ./...  --timeout 10m  --tests=false --skip-dirs tests --skip-dirs test
}


#----------------------------------------------/
# Scan containers with Anchore and Trivy       /
# Variables required are set in CI             /
#----------------------------------------------/
run_container_scan() {
	set +e 
	print_step 'Run container scan'
	COMPL_DIR=${ARTIFACTS}/compliance
	CONTAINER_SCAN_DIR=${COMPL_DIR}/container_scan
	TRIVY_LIST=${CONTAINER_SCAN_DIR}/trivy_vulnerability.txt
	TRIVY_JSON=${CONTAINER_SCAN_DIR}/trivy_vulnerability.json
	mkdir -p ${CONTAINER_SCAN_DIR}
	
	for EACH_IMAGE in ${IMAGES_LIST}
	do
		trivy --download-db-only
 		echo "Image Name: ${EACH_IMAGE} " >> ${TRIVY_LIST}
		trivy ${EACH_IMAGE} >> ${TRIVY_LIST}
 	        trivy -f json ${EACH_IMAGE} >> ${TRIVY_JSON}
	done

	gsutil cp -r $TRIVY_LIST gs://capi-prow-artifacts/compliance/$DATE/container/${REPO_NAME}.txt
	gsutil cp -r $TRIVY_JSON gs://capi-prow-artifacts/compliance/$DATE/container/${REPO_NAME}.json
	set -e 
}

#----------------------------------------------/
# Check if the release has already been        /
# done with the same version                   /
#----------------------------------------------/
check_pre_released() {

	set +e 
	set_release_vars

	gsutil ls ${VERSION_DIR}/${MARKER_FILE} 
	if [[ $? -eq 0 ]]; then	
		echo "Version ${PROD_VERSION} has already been released and is available in release folder"
		exit 1
	fi
	set -e

}

#----------------------------------------------/
# Copy manifest files for this release         /
#  Also update the latest-version.txt file     /
#----------------------------------------------/
create_release_manifest() {

	print_step "Copy manifests to release folder"

	set_release_vars

	echo 'released'      > ${MARKER_FILE}

	gsutil cp -r config ${VERSION_DIR}/kustomize
	gsutil cp -r maas-manifest.yaml  ${VERSION_DIR}/
	gsutil cp    ${MARKER_FILE}   ${VERSION_DIR}/

}


export REPO_NAME=cluster-api-provider-maas
set_image_tag
export IMG=${DOCKER_REGISTRY}/${IMG_LOC}/cluster-api-provider-maas:${IMG_TAG}
IMAGES_LIST="${IMG}"

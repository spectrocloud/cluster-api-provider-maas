#!/usr/bin/bash

set -euox pipefail

if [ "$#" -eq 0 ]; then
    echo "No kubernetes version supplied, exiting"
fi


for arg; do
  printf 'Begin image generation for k8s version "%s"\n' "$arg"
  export K8S_VERSION=$arg

  export K8S_DEB_VERSION="${K8S_VERSION}-00"

  if [[ "${K8S_VERSION}" =~ ^[0-9]\.[0-9]{0,3}\.[0-9]{0,3}$ ]]; then
      echo "Valid k8s version"
  else
      echo "Invalid k8s version"
      exit 1
  fi

  export K8S_SERIES=`echo "${K8S_VERSION}" | awk 'BEGIN {FS="."; OFS="."} {print $1,$2}'`
  export IMAGE_K8S_VERSION=`echo "${K8S_VERSION}" | awk 'BEGIN {FS="."; OFS=""} {print $1,$2,$3}'`
  export IMAGE_OS_VERSION="18045"
  export OUTPUT_OS_VERSION="1804"

  cat ./packer/qemu/qemu-ubuntu-1804.json | jq ". + {\"kubernetes_deb_version\": \"${K8S_DEB_VERSION}\", \"kubernetes_semver\": \"v${K8S_VERSION}\", \"kubernetes_series\": \"v${K8S_SERIES}\"}" > ./packer/qemu/qemu-ubuntu-1804.json.tmp
  mv ./packer/qemu/qemu-ubuntu-1804.json.tmp ./packer/qemu/qemu-ubuntu-1804.json

  cat ./packer/qemu/qemu-ubuntu-1804.json

  make deps-qemu
  export PACKER_LOG=1
  PATH=$PATH make build-qemu-ubuntu-1804



  TMP_DIR=$(mktemp -d /tmp/packer-maas-XXXX)
  echo 'Binding packer qcow2 image output to nbd ...'
  modprobe nbd
  qemu-nbd -d /dev/nbd4
  qemu-nbd -c /dev/nbd4 -n ./output/ubuntu-${OUTPUT_OS_VERSION}-kube-v${K8S_VERSION}/ubuntu-${OUTPUT_OS_VERSION}-kube-v${K8S_VERSION}
  echo 'Waiting for partitions to be created...'
  tries=0
  while [ ! -e /dev/nbd4p1 -a $tries -lt 60 ]; do                                                                                                             
      sleep 1
      tries=$((tries+1))                                                                                                                                  
  done
  echo "mounting image..."
  mount /dev/nbd4p1 $TMP_DIR
  echo 'Tarring up image...'
  tar -Sczpf u-${OUTPUT_OS_VERSION}-0-k-${IMAGE_K8S_VERSION}-0.tar.gz --selinux -C $TMP_DIR  .                                                                                            
  echo 'Unmounting image...'
  umount $TMP_DIR                                                                                                                                         
  qemu-nbd -d /dev/nbd4
  rmdir $TMP_DIR



  mkdir -p $HOME/.aws

  if [[ -z "${AWS_ACCESS_KEY_ID}" ]]; then
    echo "aws access key id not set exiting"
    exit 1
  fi

  if [[ -z "${AWS_SECRET_ACCESS_KEY}" ]]; then
    echo "aws access key secret not set exiting"
    exit 1
  fi

  echo "[goldenci-bucket]
  aws_access_key_id = ${AWS_ACCESS_KEY_ID}
  aws_secret_access_key = ${AWS_SECRET_ACCESS_KEY}
  region =  us-east-1" > $HOME/.aws/credentials

  export AWS_PROFILE=goldenci-bucket

  aws s3api put-object --acl public-read --bucket maasgoldenimage --key "u-${OUTPUT_OS_VERSION}-0-k-${IMAGE_K8S_VERSION}-0.tar.gz" --body u-${OUTPUT_OS_VERSION}-0-k-${IMAGE_K8S_VERSION}-0.tar.gz

  echo 'image upload done'


  echo "cleaning image outputs directory"

  rm u-${OUTPUT_OS_VERSION}-0-k-${IMAGE_K8S_VERSION}-0.tar.gz
  rm -rf ./output/ubuntu-${OUTPUT_OS_VERSION}-kube-v${K8S_VERSION}/
done
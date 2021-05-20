#!/bin/bash
#
# Copyright 2021 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
# Use this script to upload images for Red Hat certification
#
# ./push-images-rh.sh <image-name> <tag> <arch> <Registry Key from RH cert> <RH PID> <SHA or none> <-test>
#
# Go to the "images" URL (https://connect.redhat.com/project/3637401/images)
# - under Container Images -> PID, copy the ospid value for <RH PID>
# - select "Push Image Manually", goto View Registry Key and copy the Registry Key for <Registry Key from RH cert>
#   the Registry Key is a very long string, over 1000 chars
#
# Use the "-test" parm the first time to upload the image with a temp tag so you can verify the image will pass the scan.
# To make the command line parameters easier to manage, put the registry-key, pid and sha in env vars
#   ./push-images-rh.sh <image-name> <tag> <arch> $REG_KEY $RH_PID $SHA <-test>
#

IMAGE_NAME=$1
TAG=$2
ARCH=$3
PASSWORD=$4
RH_PID=$5
# have to pull operator arch images by SHA or build tag (v20210413-584c257).
# get SHAs from list.manifest.json in the multi-arch image
SHA=$6
TEST=$7

if [[ $SHA == "" ]]; then
   echo "Missing parm. Need <image-name> <image-tag> <arch> <Registry Key from RH cert> <Red Hat project PID> <SHA or none>"
   echo "Example:"
   echo "   push-images-rh.sh ibm-management-ingress-operator 1.5.1 amd64 REG_KEY RH_PID SHA"
   exit 1
fi

# check the ARCH value
if [[ $ARCH != "amd64" && $ARCH != "ppc64le" && $ARCH != "s390x" ]]; then
   echo "$ARCH is not valid. ARCH must be amd64, ppc64le, or s390x"
   exit 1
fi

# "none" means pull by image tag instead of SHA.
# if the parm starts with "sha256", it's a SHA so use it
# anything else will be considered to be a build tag like v20210413-584c257
if [[ $SHA == "none" ]]; then
   IMAGE_SUFFIX=:$TAG
elif [[ $SHA =~ "sha256" ]]; then
   IMAGE_SUFFIX=@$SHA
else
   IMAGE_SUFFIX=:$SHA
fi
echo "IMAGE_SUFFIX=$IMAGE_SUFFIX"


#OLD: QUAY=quay.io/opencloudio/$IMAGE:$TAG

LOCAL_IMAGE=hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com/ibmcom/$IMAGE_NAME-$ARCH$IMAGE_SUFFIX
#OLD LOCAL_IMAGE=hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com/ibmcom/$IMAGE_NAME-$ARCH@$SHA
#OLD LOCAL_IMAGE=hyc-cloud-private-integration-docker-local.artifactory.swg-devops.com/ibmcom/$IMAGE_NAME-$ARCH:$TAG
SCAN_IMAGE=scan.connect.redhat.com/$RH_PID/$IMAGE_NAME:$TAG-$ARCH$TEST

echo "### This is going to pull your image from the repo"
echo docker pull "$LOCAL_IMAGE"
echo
echo "### This will log you into your RH project for THIS image"
echo docker login -u unused -p "$PASSWORD" scan.connect.redhat.com
echo
echo "### This will tag the image for RH scan"
echo docker tag "$LOCAL_IMAGE" "$SCAN_IMAGE"
echo
echo "### This will push to Red Hat... if this is the first scan there MUST be something appended to the end."
echo docker push "$SCAN_IMAGE"

echo
echo
echo "Does the above look correct? (y/n) "
read -r ANSWER
if [[ "$ANSWER" != "y" ]]
then
  echo "Not going to run commands"
  exit 1
fi

docker pull "$LOCAL_IMAGE"
docker login -u unused -p "$PASSWORD" scan.connect.redhat.com
docker tag "$LOCAL_IMAGE" "$SCAN_IMAGE"
docker push "$SCAN_IMAGE"

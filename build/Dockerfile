#
# Copyright 2020 IBM Corporation
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

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.1-328
ARG VCS_REF
ARG VCS_URL

LABEL org.label-schema.vendor="IBM" \
  org.label-schema.name="management-ingress-operator" \
  org.label-schema.description="Management Ingress Operator" \
  org.label-schema.vcs-ref=$VCS_REF \
  org.label-schema.vcs-url=$VCS_URL \
  org.label-schema.license="Licensed Materials - Property of IBM" \
  org.label-schema.schema-version="1.0" \
  name="management-ingress-operator" \
  vendor="IBM" \
  description="Management Ingress Operator" \
  summary="Management Ingress Operator"

ENV OPERATOR=/usr/local/bin/management-ingress-operator \
  DEPLOY_DIR=/deploy \
  USER_UID=1001 \
  USER_NAME=management-ingress-operator

# install operator binary
COPY build/_output/bin/management-ingress-operator ${OPERATOR}
COPY deploy/crds ${DEPLOY_DIR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

# copy licenses
RUN mkdir /licenses
COPY LICENSE /licenses

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}

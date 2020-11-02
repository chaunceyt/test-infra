# Copyright 2020 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# set default shell
SHELL=/bin/bash -o pipefail

REGISTRY ?= local
IMGNAME = flake-tracker
IMAGE_VERSION ?= v0.0.1

IMAGE = $(REGISTRY)/$(IMGNAME)

TAG ?= $(shell git describe --tags --always --dirty)

# Build args
GO_VERSION ?= 1.15.3
DISTROLESS_IMAGE ?= static-debian10

GO_BUILD ?= go build

BUILD_ARGS = --build-arg=GO_VERSION=$(GO_VERSION) \
             --build-arg=DISTROLESS_IMAGE=$(DISTROLESS_IMAGE)

.PHONY: all build clean

.PHONY: all
all: build

.PHONY: build
build:
	$(GO_BUILD)

.PHONY: clean
clean:
	rm flake-tracker

.PHONY: container
container:
	echo "Building $(IMGNAME)."
	docker build \
			--tag $(IMAGE):$(IMAGE_VERSION) \
			--tag $(IMAGE):$(TAG) \
			--tag $(IMAGE):latest \
			$(BUILD_ARGS) \
			-f $(CURDIR)/Dockerfile .

.PHONY: push
push: container
	echo "Pushing $(IMGNAME) tags"
	docker push $(IMAGE):$(IMAGE_VERSION); \
	docker push $(IMAGE):$(TAG); \
	docker push $(IMAGE):latest;

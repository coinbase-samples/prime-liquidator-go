# Copyright 2024-present Coinbase Global, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#  http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

REGION ?= us-east-1
PROFILE ?= sa-infra
ENV_NAME ?= dev

BUILD_ID ?= latest

STACK_NAME ?= prime-liquidator-$(ENV_NAME)
ACCOUNT_ID := $(shell aws sts get-caller-identity --profile $(PROFILE) --query 'Account' --output text)

.PHONY: create-aws-stack
create-aws-stack:
	@aws cloudformation create-stack \
	--profile $(PROFILE) \
	--stack-name $(STACK_NAME) \
	--region $(REGION) \
	--capabilities CAPABILITY_NAMED_IAM \
	--template-body file://infra/aws.cfn.yml \
	--parameters file://infra/aws-$(ENV_NAME).json

.PHONY: update-aws-stack
update-aws-stack:
	@aws cloudformation update-stack \
	--profile $(PROFILE) \
	--stack-name $(STACK_NAME) \
	--region $(REGION) \
	--capabilities CAPABILITY_NAMED_IAM \
	--template-body file://infra/aws.cfn.yml \
	--parameters file://infra/aws-$(ENV_NAME).json

.PHONY: delete-aws-stack
delete-aws-stack:
	@aws cloudformation delete-stack --profile $(PROFILE) --stack-name $(STACK_NAME) --region $(REGION)

.PHONY: validate-aws-template
validate-aws-template:
	@aws cloudformation validate-template --profile $(PROFILE) --template-body file://infra/aws.cfn.yml --region $(REGION) --no-cli-pager

.PHONY: ecr-public-login
ecr-public-login:
	@aws ecr-public get-login-password --region $(REGION) --profile $(PROFILE) | docker login --username AWS --password-stdin public.ecr.aws

.PHONY: build-image
build-image:
	@aws ecr-public get-login-password --region $(REGION) --profile $(PROFILE) | docker login --username AWS --password-stdin public.ecr.aws
	@aws ecr get-login-password --region $(REGION) --profile $(PROFILE) | docker login --username AWS --password-stdin $(ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com
	@docker build --tag $(STACK_NAME):$(BUILD_ID) . -f ./Dockerfile
	@docker tag $(STACK_NAME):$(BUILD_ID) $(ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com/$(STACK_NAME):$(BUILD_ID)
	@docker push $(ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com/$(STACK_NAME):$(BUILD_ID)


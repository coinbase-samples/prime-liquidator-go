REGION ?= us-east-1
PROFILE ?= sa-infra
ENV_NAME ?= dev

STACK_NAME ?= prime-liq-$(ENV_NAME)

CLUSTER ?= $(STACK_NAME)

ACCOUNT_ID := $(shell aws sts get-caller-identity --profile $(PROFILE) --query 'Account' --output text)

.PHONY: create-stack
create-stack:
	@aws cloudformation create-stack \
	--profile $(PROFILE) \
	--stack-name $(STACK_NAME) \
	--region $(REGION) \
	--capabilities CAPABILITY_NAMED_IAM \
	--template-body file://poc.cfn.yml \
	--parameters file://poc.json

.PHONY: delete-stack
delete-stack:
	@aws cloudformation delete-stack \
  --profile $(PROFILE) \
  --stack-name $(STACK_NAME) \
  --region $(REGION)

.PHONY: validate-template
validate-template:
	@aws cloudformation validate-template \
  --profile $(PROFILE) \
  --template-body file://poc.cfn.yml \
  --region $(REGION)

.PHONY: update-stack
update-stack:
	@aws cloudformation update-stack \
  --profile $(PROFILE) \
  --stack-name $(STACK_NAME) \
  --region $(REGION) \
  --capabilities CAPABILITY_NAMED_IAM \
  --template-body file://poc.cfn.yml \
	--parameters file://poc.json

PHONY: update-service
update-service:
	@aws ecr get-login-password \
  --profile $(PROFILE) \
  --region $(REGION) \
	| docker login --username AWS --password-stdin $(ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com
	@docker build -t $(STACK_NAME) .
	@docker tag $(STACK_NAME):latest $(ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com/$(STACK_NAME):latest
	@docker push $(ACCOUNT_ID).dkr.ecr.$(REGION).amazonaws.com/$(STACK_NAME):latest
	@aws ecs update-service \
 	--no-cli-pager \
	--profile $(PROFILE) \
	--region $(REGION) \
	--cluster $(CLUSTER) \
  --service $(STACK_NAME) \
  --force-new-deployment

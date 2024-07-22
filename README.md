# Coinbase Prime Liquidator


[![GoDoc](https://godoc.org/github.com/coinbase-samples/prime-liquidator-go?status.svg)](https://godoc.org/github.com/coinbase-samples/prime-liquidator-go)
[![Go Report Card](https://goreportcard.com/badge/coinbase-samples/prime-liquidator-go)](https://goreportcard.com/report/coinbase-samples/prime-liquidator-go)

## Overview

The *Coinbase Prime Liquidator* sample application continuously monitors a [Coinbase Prime](https://prime.coinbase.com/) portfolio
for crypto assets in hot/trading wallets and places sell orders or converts to USD/fiat.

Sell orders deduct holds based on instrument, so if new assets are added while others are being liquidated, the
application continues to create new orders if there are enough assets to sell. Additionally, if for some reason an order
continuously fails to execute, there is an ID generated (client_order_id) from the sell attributes that is used to
reduce the amount of spam/failing orders sent to Prime. Prime treats the client_order_id as idempotent for open orders and
the ID is cached in-process too.

## License

The *Coinbase Prime Liquidator* sample application is free and open source and released under the [Apache License, Version 2.0](LICENSE).

The application and code are only available for demonstration purposes.

## Warning

**Use of this sample application may cause a negative financial impact**

When the application is running, it continuously monitors and converts crypto assets to USD.
If the application is accidentally left running or mistakenly pointed at an unintended portfolio,
all of the assets in hot/trading wallets will be quickly liquidated.

Sell orders are created with a one hour TWAP with the lowest tolerance (limit price) set at 10% below the
current price of the asset price on the [Coinbase Exchange](https://exchange.coinbase.com/).

If the sample application is used to liquidate large positions, there is price action risk that may
result in trades executing up to 10% lower than the latest price check.

## Usage

### Create Stack

The *Coinbase Prime Liquidator* has a [sample AWS CloudFormation](infra/aws.cfn.yml) (CFN) template that can be deployed to run the application in an Amazon Elastic Container Service (Amazon ECS) cluster. This template creates all of the required resources and can be customized to suit the deployers needs. To deploy the CFN stack, [initialize your AWS credentials](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html) and then run:

 ```bash
make create-aws-stack ENV_NAME=dev PROFILE=default REGION=us-east-1
```

Customize the values of the *ENV_NAME*, *PROFILE*, and *REGION* to match the needs of your environment. The *PROFILE* argument is the name of the AWS CLI profile configured and the *REGION* argument is the [AWS Region](https://aws.amazon.com/about-aws/global-infrastructure/regions_az/) you would like to deploy to.

### Prime Credentials

Once the CFN stack is deployed, configure your credentials in [AWS Secrets Manager](https://docs.aws.amazon.com/secretsmanager/latest/userguide/intro.html). The name of the empty secret uses the following format:

```
prime-liquidator-ENV_NAME-prime-api-credentials
```

The *ENV_NAME* will the same as what was passed to the *create-aws-stack* command (e.g., dev). The credentials use the following format:

```
{
  "accessKey": "",
  "passphrase": "",
  "signingKey": "",
  "portfolioId": "",
  "svcAccountId": ""
}
```

Prime API credentials can be created in the [Prime web application](https://prime.coinbase.com), once an account is opened.

### Build/Deploy Container

The inital stack deploys the *public.ecr.aws/nginx/nginx:stable-perl-arm64v8* container as a placeholder. The ECS task definition/service requires a container and at this point, the ECR repository has not been created. Once the CFN stack is deployed, build the container image and deploy to the Amazon ECR (ECR) repository. To do this, execute:

 ```bash
make build-image ENV_NAME=dev PROFILE=default REGION=us-east-1 BUILD_ID=1
```

Again, customize the values of the *ENV_NAME*, *PROFILE*, *BUILD_ID*, and *REGION* to match the needs of your environment. The *BUILD_ID* can be set to a value that matches your container tagging practices.

After the build is deployed, update the *DockerImageUri* attribute in the [aws-dev.json](infra/aws-dev.json) file to the URI of your deployed build. The newly created ECR respository uses the following format:

```
AWS_ACCOUNT_ID.dkr.ecr.AWS_REGION.amazonaws.com/prime-liquidator-ENV_NAME:BUILD_ID
```

Note: If your environment is not named, *dev*, you will need to create a new environment configuration file. The environment name is defined by the *ENV_NAME* CLI argument. The format for the environment configuration file name is:

```
infra/aws-ENV_NAME.json
```

The CFN stack is configured to run ARM64 containers, so you must build on an ARM64 compatible computer.

### Update Stack

Once the *DockerImageUri* attribute is updated, run the following command to update the CFN stack and run the *Coinbase Prime Liquidator*:

 ```bash
make update-aws-stack ENV_NAME=dev PROFILE=default REGION=us-east-1
```

This command deploys the container image specified in the *DockerImageUri* and starts listening for new Prime activities.

## Building

To build the sample application, ensure that [Go](https://go.dev/) 1.21+ is installed and then run:

```bash
go build cmd/server/main.go
```

To build the Docker container, login to the [Amazon ECR Public Gallery](https://gallery.ecr.aws/):

```bash
aws ecr-public get-login-password --region us-east-1 | docker login --username AWS --password-stdin public.ecr.aws
```

Run the docker build:

```bash
docker build .
```



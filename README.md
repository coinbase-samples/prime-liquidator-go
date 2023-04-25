# Liquidator README

## Overview

The *Liquidator* sample application continuously monitors a [Coinbase Prime](https://prime.coinbase.com/) portfolio
for crypto assets in hot/trading wallets and places sell orders or converts to USD/fiat.

Sell orders deduct holds based on instrument, so if new assets are added while others are being liquidated, the
application continues to create new orders if there are enough assets to sell. Additionally, if for some reason an order
continuously fails to execute, there is an ID generated (client_order_id) from the sell attributes that is used to
reduce the amount of spam/failing orders sent to Prime. Prime treats the client_order_id as idempotent for open orders and
the ID is is cached in-process too.

## License

The *Liquidator* sample application is free and open source and released under the [Apache License, Version 2.0](LICENSE).

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

## Building

To build the sample application, ensure that [Go](https://go.dev/) 1.19+ is installed and then run:

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


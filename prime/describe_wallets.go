package prime

import (
	"context"
	"fmt"
)

func DescribeWallets(
	ctx context.Context,
	request *DescribeWalletsRequest,
) (*DescribeWalletsResponse, error) {

	url := fmt.Sprintf("%s/portfolios/%s/wallets?type=%s",
		primeV1ApiBaseUrl,
		request.PortfolioId,
		request.Type,
	)

	url = urlIteratorParams(url, request.IteratorParams)

	for _, v := range request.Symbols {
		url += fmt.Sprintf("&symbols=%s", v)
	}

	response := &DescribeWalletsResponse{Request: request}

	if err := PrimeGet(ctx, url, request, response); err != nil {
		return nil, fmt.Errorf("unable to DescribeWallets: %w", err)
	}

	return response, nil
}

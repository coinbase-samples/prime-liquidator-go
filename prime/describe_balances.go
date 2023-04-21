package prime

import (
	"context"
	"fmt"
)

type DescribeBalancesRequest struct {
	PortfolioId string   `json:"portfolio_id"`
	Type        string   `json:"balance_type"`
	Symbols     []string `json:"symbols"`
}

type DescribeBalancesResponse struct {
	Balances        []*AssetBalances         `json:"balances"`
	Type            string                   `json:"type"`
	TradingBalances *BalanceWithHolds        `json:"trading_balances"`
	VaultBalances   *BalanceWithHolds        `json:"vault_balances"`
	Request         *DescribeBalancesRequest `json:"request"`
}

func DescribeBalances(
	ctx context.Context,
	request *DescribeBalancesRequest,
) (*DescribeBalancesResponse, error) {

	url := fmt.Sprintf("%s/portfolios/%s/balances", primeV1ApiBaseUrl, request.PortfolioId)

	var appended bool
	if len(request.Type) > 0 {
		url += fmt.Sprintf("?balance_type=%s", request.Type)
		appended = true
	}

	for _, v := range request.Symbols {
		url += fmt.Sprintf("%ssymbols=%s", urlParamSep(appended), v)
		appended = true
	}

	response := &DescribeBalancesResponse{Request: request}

	if err := PrimeGet(ctx, url, request, response); err != nil {
		return nil, fmt.Errorf("unable to DescribeBalances: %w", err)
	}

	return response, nil

}

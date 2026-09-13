package financialitems

// FakeCreateFinancialItemRequest returns a deterministic, public-safe financial item fixture.
func FakeCreateFinancialItemRequest() CreateFinancialItemRequest {
	return CreateFinancialItemRequest{
		Name:                        "Example brokerage account",
		AmountCents:                 1250000,
		Currency:                    "USD",
		AnnualReturnRateBasisPoints: 700,
		AnnualContributionCents:     300000,
		SortOrder:                   10,
	}
}

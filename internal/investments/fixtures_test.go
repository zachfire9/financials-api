package investments

// FakeCreateInvestmentRequest returns a deterministic, public-safe investment fixture.
func FakeCreateInvestmentRequest() CreateInvestmentRequest {
	return CreateInvestmentRequest{
		Name:                    "Example taxable brokerage",
		Type:                    InvestmentTypeTaxableBrokerage,
		Institution:             "Example Institution",
		BalanceCents:            1250000,
		Currency:                "USD",
		AnnualContributionCents: 300000,
	}
}

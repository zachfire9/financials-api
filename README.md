# Financials API

## Logic

- Pass in the request info
- Loop through every time period, add contribution amount, and apply return rate to current total amount
- Add inflation to retirement annual spending amount every year
- Stop when you hit total years before retirement
- Get the total amount for all the years of each investment
- Subtract annual spending amount from total while total amount is greater than annual spending amount 
- Keep track of number of years before depleted

## Requests

### Plan

```
{
	"yearsBefore": "30",
	"annualSpending": "30000",
	"annualInflation": ".03",
	"returnRate": ".03"
	"investments": [Investment]
}
```

### Investment

```
{
	"amount": "1000",
	"contribution": "200",
	"returnRate": ".05"
}
```
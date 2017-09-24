/**
 * 
 */
package com.financials.api;

import java.math.BigDecimal;
import java.util.ArrayList;

/**
 * @author Zach Firestone
 *
 */
public class Retirement {
	
	private Plan plan;
	private BigDecimal total;
	private BigDecimal annualSpendingInflation;
	private ArrayList<Year> years;
	
	public Retirement(Plan plan) {
		this.plan = plan;
	}

	public void calculate() {
		
		ArrayList<Investment> investments = plan.getInvestments();
		this.years  = new ArrayList<Year>();
		this.total = new BigDecimal("0.00");
		this.annualSpendingInflation = new BigDecimal(plan.getAnnualSpending());
		
		for (int i = 0; i < Integer.parseInt(plan.getYearsBefore()); i++) {
			Year year = new Year();
			ArrayList<Investment> yearInvestments = new ArrayList<Investment>();
			BigDecimal yearInvestmentAmount = new BigDecimal("0.00");
			for (int j = 0; j < investments.size(); j++) {
				Investment yearInvestment = new Investment();
				yearInvestment.setContribution(investments.get(j).getContribution());
				yearInvestment.setReturnRate(investments.get(j).getReturnRate());
				
				BigDecimal contributionAmount = new BigDecimal(investments.get(j).getContribution());
				BigDecimal returnRate = new BigDecimal(investments.get(j).getReturnRate());
				
				if (i > 0) {
					// Get the last years investments and then get the same index that we're currently on
					Investment previousYearsInvestment = this.years.get(i - 1).getInvestments().get(j);
					BigDecimal newInvestmentAmount = new BigDecimal(previousYearsInvestment.getAmount()).add(contributionAmount);
					yearInvestment.setAmount(newInvestmentAmount.add(newInvestmentAmount.multiply(returnRate)).setScale(2, BigDecimal.ROUND_HALF_UP).toString());
				} else {
					BigDecimal newInvestmentAmount = new BigDecimal(investments.get(j).getAmount()).add(contributionAmount);
					yearInvestment.setAmount(newInvestmentAmount.add(newInvestmentAmount.multiply(returnRate)).setScale(2, BigDecimal.ROUND_HALF_UP).toString());
				}
				
				yearInvestmentAmount = yearInvestmentAmount.add(new BigDecimal(yearInvestment.getAmount()));
				yearInvestments.add(yearInvestment);
				
				if (i == (Integer.parseInt(plan.getYearsBefore()) - 1)) {
					this.total = this.total.add(new BigDecimal(yearInvestment.getAmount())).setScale(2, BigDecimal.ROUND_HALF_UP);
				}
			}

			year.setAmount(yearInvestmentAmount.setScale(2, BigDecimal.ROUND_HALF_UP).toString());
			year.setInvestments(yearInvestments);
			year.setAnnualSpendingInflation(this.annualSpendingInflation.toString());
			this.years.add(year);
			this.annualSpendingInflation = this.annualSpendingInflation.add(this.annualSpendingInflation.multiply(new BigDecimal(plan.getAnnualInflation()))).setScale(2, BigDecimal.ROUND_HALF_UP);
		}
		
		while (this.total.compareTo(BigDecimal.ZERO) > 0) {
			Year year = new Year();
			Investment yearInvestment = new Investment();
			
			yearInvestment.setContribution("0.00");
			yearInvestment.setReturnRate(plan.getReturnRate());
			
			this.annualSpendingInflation = this.annualSpendingInflation.add(this.annualSpendingInflation.multiply(new BigDecimal(plan.getAnnualInflation()))).setScale(2, BigDecimal.ROUND_HALF_UP);
			
			this.total = this.total.subtract(this.annualSpendingInflation);
			yearInvestment.setAmount(this.total.toString());

			if (this.total.setScale(2, BigDecimal.ROUND_HALF_UP).compareTo(BigDecimal.ZERO) > 0) {
				year.setAmount(this.total.setScale(2, BigDecimal.ROUND_HALF_UP).toString());
				ArrayList<Investment> yearInvestments = new ArrayList<Investment>();
				yearInvestments.add(yearInvestment);
				year.setInvestments(yearInvestments);
				year.setAnnualSpendingInflation(this.annualSpendingInflation.toString());
				this.years.add(year);
			} else {
				this.total = new BigDecimal("0.00");
			}
		}
	}

	/**
	 * @return the plan
	 */
	public Plan getPlan() {
		return plan;
	}

	/**
	 * @param plan the plan to set
	 */
	public void setPlan(Plan plan) {
		this.plan = plan;
	}

	/**
	 * @return the years
	 */
	public ArrayList<Year> getYears() {
		return years;
	}

	/**
	 * @param years the years to set
	 */
	public void setYears(ArrayList<Year> years) {
		this.years = years;
	}
}

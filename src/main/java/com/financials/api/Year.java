/**
 * 
 */
package com.financials.api;

import java.io.Serializable;
import java.math.BigDecimal;
import java.util.ArrayList;

/**
 * @author a206473540
 *
 */
public class Year implements Serializable {

	private String amount;
	private String annualSpendingInflation;
	private ArrayList<Investment> investments;
	
	/**
	 * @return the amount
	 */
	public String getAmount() {
		return amount;
	}

	/**
	 * @param amount the amount to set
	 */
	public void setAmount(String amount) {
		this.amount = amount;
	}

	/**
	 * @return the annualSpendingInflation
	 */
	public String getAnnualSpendingInflation() {
		return annualSpendingInflation;
	}

	/**
	 * @param annualSpendingInflation the annualSpendingInflation to set
	 */
	public void setAnnualSpendingInflation(String annualSpendingInflation) {
		this.annualSpendingInflation = annualSpendingInflation;
	}

	/**
	 * @return the investments
	 */
	public ArrayList<Investment> getInvestments() {
		return investments;
	}

	/**
	 * @param investments the investments to set
	 */
	public void setInvestments(ArrayList<Investment> investments) {
		this.investments = investments;
	}
}

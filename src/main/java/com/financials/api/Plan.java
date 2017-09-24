/**
 * 
 */
package com.financials.api;

import java.io.Serializable;
import java.util.ArrayList;

/**
 * @author Zach Firestone
 *
 */
public class Plan implements Serializable {

	private String yearsBefore;
	private String annualSpending;
	private String annualInflation;
	private String returnRate;
	private ArrayList<Investment> investments;

	/**
	 * @return the yearsBefore
	 */
	public String getYearsBefore() {
		return yearsBefore;
	}

	/**
	 * @param yearsBefore the yearsBefore to set
	 */
	public void setYearsBefore(String yearsBefore) {
		this.yearsBefore = yearsBefore;
	}

	/**
	 * @return the annualSpending
	 */
	public String getAnnualSpending() {
		return annualSpending;
	}

	/**
	 * @param annualSpending the annualSpending to set
	 */
	public void setAnnualSpending(String annualSpending) {
		this.annualSpending = annualSpending;
	}

	/**
	 * @return the annualInflation
	 */
	public String getAnnualInflation() {
		return annualInflation;
	}

	/**
	 * @param annualInflation the annualInflation to set
	 */
	public void setAnnualInflation(String annualInflation) {
		this.annualInflation = annualInflation;
	}

	/**
	 * @return the returnRate
	 */
	public String getReturnRate() {
		return returnRate;
	}

	/**
	 * @param returnRate the returnRate to set
	 */
	public void setReturnRate(String returnRate) {
		this.returnRate = returnRate;
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

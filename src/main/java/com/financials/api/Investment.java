package com.financials.api;

import java.io.Serializable;

public class Investment implements Serializable {
	
	private String amount;
	private String contribution;
	private String returnRate;

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
	 * @return the contribution
	 */
	public String getContribution() {
		return contribution;
	}

	/**
	 * @param contribution the contribution to set
	 */
	public void setContribution(String contribution) {
		this.contribution = contribution;
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
	
}

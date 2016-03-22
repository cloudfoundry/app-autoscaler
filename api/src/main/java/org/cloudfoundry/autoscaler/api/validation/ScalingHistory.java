package org.cloudfoundry.autoscaler.api.validation;

import java.util.List;

import javax.validation.Valid;

public class ScalingHistory {
	@Valid
	private List<HistoryData> data;

	public List<HistoryData> getData() {
		return this.data;
	}

	public void setData(List<HistoryData> data) {
		this.data = data;
	}

}
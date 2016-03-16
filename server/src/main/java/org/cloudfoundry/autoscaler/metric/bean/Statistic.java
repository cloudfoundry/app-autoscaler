package org.cloudfoundry.autoscaler.metric.bean;

import org.cloudfoundry.autoscaler.bean.Condition.AggregationType;

public class Statistic {
	
	private final AggregationType type;
	
	private int count = 0;
	private double curValue = 0;
	
	public Statistic(AggregationType type) {
		this.type = type;
	}
	
	public void update(double value) {
		switch (type) {
		case AVG:
			double sum = curValue * count;
			sum += value;
			count++;
			this.curValue = sum / count;
			break;
		default:
			if (value > curValue)
				curValue = value;
			count = 1;
			break;
		}
	}
	
	public double getValue() {
		return this.curValue;
	}
	
	public int getCount() {
		return this.count;
	}
	

}

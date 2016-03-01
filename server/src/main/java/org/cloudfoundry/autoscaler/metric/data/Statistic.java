package org.cloudfoundry.autoscaler.metric.data;

import org.cloudfoundry.autoscaler.cloudservice.api.monitor.Condition.AggregationType;

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
	
	/*public void update(Statistic stat) throws DifferentStatisticTypeException {
		
		if (stat.type != this.type) {
			throw new DifferentStatisticTypeException();
		}
		
		switch (this.type) {
		case STAT_TYPE_AVG:
			break;
		default:
			break;
		}
	}*/

}

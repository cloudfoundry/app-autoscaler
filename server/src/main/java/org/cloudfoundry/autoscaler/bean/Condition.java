package org.cloudfoundry.autoscaler.bean;

/**
 * Defines a condition that can be concatenated as a boolean expression in Trigger for joint evaluation
 * Note: 	1) Conditions are evaluated according to their nature order in conditionList of Trigger;
 * 			2) No association is supported at this point.
 * 
 *
 */
public class Condition {
	
	public static enum AggregationType {
		AVG, MAX, MIN
	}
	
	public static enum ThresholdType {
		LARGER_THAN, SMALLER_THAN
	}
	
	public static enum ConnectiveType {
		AND, OR
	}
	
	public	String metricId						= "";
	public AggregationType windowStatType     	= AggregationType.AVG;
	public AggregationType instanceStatType   	= AggregationType.MAX;
	public int statWindowSecs	     			= 60;
	public int breachDurationSecs 				= 60;
	public int metricThreshold    				= 70;
	public ThresholdType thresholdType      	= ThresholdType.LARGER_THAN;
	
	// defines how this condition should be connected with the rest of the conditions defined in a trigger
	public ConnectiveType connectiveType = ConnectiveType.AND;

}

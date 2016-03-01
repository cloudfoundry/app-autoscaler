package org.cloudfoundry.autoscaler.util;

import java.util.HashMap;
import java.util.Map;
/**
 * This class is used to do mapping between autoscaling service and monitor service
 * 
 *
 */
public class IcapMonitorMetricsMapper {
	private static Map<String, String> nameMapper = new HashMap<String, String>();
	static{
		nameMapper.put("CPUUTILIZATION", "CPU");//CPUUtilzation
		nameMapper.put("MEMORY", "Memory");//Memory
	}
	
	/**
	 * Mapping metric name
	 * @return
	 */
	public static Map<String, String> getMetricNameMapper(){
		return nameMapper;
	}
	
	/**
	 * Convert metric value to metric value of monitor service
	 * @return
	 */
	public static double converMetricValue(String metricName, double value){

		return value;
	}
}

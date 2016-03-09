package org.cloudfoundry.autoscaler.util;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

public class IcapMonitorMetricsMapperTest {
	
	@Test
	public void IcapMonitorMetricsMappterValueTest(){
		assertEquals(IcapMonitorMetricsMapper.getMetricNameMapper().get("CPUUTILIZATION"),"CPU");
		assertEquals(IcapMonitorMetricsMapper.getMetricNameMapper().get("MEMORY"),"Memory");
		assertEquals(IcapMonitorMetricsMapper.converMetricValue("CPUUTILIZATION", 10),10,0);
	}

}

package org.cloudfoundry.autoscaler.rest;

import java.lang.management.ManagementFactory;
import java.lang.management.MemoryMXBean;
import java.lang.management.MemoryUsage;
import java.util.HashMap;
import java.util.Map;

import javax.management.MBeanServer;
import javax.management.ObjectName;
import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.Produces;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;
import javax.ws.rs.core.Response.Status;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.metric.monitor.MonitorController;

import com.fasterxml.jackson.databind.ObjectMapper;

@Path("/stats")
public class ServerStatsRestApi {
	private static final Logger logger = Logger.getLogger(ServerStatsRestApi.class);

	private final MemoryMXBean memoryMXBean = ManagementFactory.getMemoryMXBean();
	private final MBeanServer mbs = ManagementFactory.getPlatformMBeanServer();

	private final ObjectMapper mapper = new ObjectMapper();

	@GET
	@Produces(MediaType.APPLICATION_JSON)
	public Response getServerStats() {
		try {
			Map<String, Object> stats = new HashMap<String, Object>();

			MemoryUsage heap = memoryMXBean.getHeapMemoryUsage();
			MemoryUsage nonHeap = memoryMXBean.getNonHeapMemoryUsage();
			stats.put("heap", heap);
			stats.put("nonHeap", nonHeap);
			double processCpuLoad = 0;

			processCpuLoad = (Double) mbs.getAttribute(new ObjectName("java.lang:type=OperatingSystem"),
					"ProcessCpuLoad");
			stats.put("processCpuLoad", processCpuLoad);

			Map<String, Integer> appstatsMap = MonitorController.getInstance().getBoundAppStats();
			stats.put("appCount", appstatsMap.get("appCount"));
			stats.put("instanceCount", appstatsMap.get("instanceCount"));

			return RestApiResponseHandler.getResponseOk(mapper.writeValueAsString(stats));

		} catch (Exception e) {
			logger.error(e.getMessage(), e);
			return RestApiResponseHandler.getResponse(Status.INTERNAL_SERVER_ERROR);
		}

	}

}

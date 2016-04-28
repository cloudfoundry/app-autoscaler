package org.cloudfoundry.autoscaler.api.rest.mock.serverapi;

import static org.cloudfoundry.autoscaler.api.Constants.TESTAPPID;
import static org.cloudfoundry.autoscaler.api.Constants.TESTAPPNAME;
import static org.cloudfoundry.autoscaler.api.Constants.TESTSERVICEID;

import javax.ws.rs.GET;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.Produces;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

import org.cloudfoundry.autoscaler.api.util.RestApiResponseHandler;

@Path("/services")
public class ServerMetricsRestApi {

	

	@GET
	@Path("/metrics/{serviceId}/{appId}")
	@Produces(MediaType.APPLICATION_JSON)
	public Response getAppHistoryMetrics(@PathParam("serviceId") String serviceId, @PathParam("appId") String appId,
			@QueryParam("newerThan") long newerThan, @QueryParam("timeRange") long timeRange) {

		String metricStr = "{\"appId\":\"" + TESTAPPID + "\",\"appName\":\"" + TESTAPPNAME
				+ "\",\"appType\":\"nodejs\",\"instanceMetrics\":[{\"appId\":\"" + TESTAPPID + "\",\"appName\":\""
				+ TESTAPPNAME + "\",\"appType\":\"nodejs\",\"serviceId\":\"" + TESTSERVICEID
				+ "\",\"instanceIndex\":0,\"timestamp\":1459331294985,\"instanceId\":\"f10be45d28b3418e8a31a86421f24679\",\"metrics\":[{\"name\":\"ProcessCpuLoad\",\"value\":\"0.0354114725\",\"category\":\"nodejs\",\"group\":\"ProcessCpuLoad\",\"timestamp\":1459331294985,\"unit\":\"%%\",\"desc\":\"\"},{\"name\":\"memory\",\"value\":\"32192716.8\",\"category\":\"nodejs\",\"group\":\"memory\",\"timestamp\":1459331294985,\"unit\":\"Bytes\",\"desc\":\"\"},{\"name\":\"init\",\"value\":\"0\",\"category\":\"nodejs\",\"group\":\"HeapMemoryUsage\",\"timestamp\":1459331294985,\"unit\":\"Bytes\",\"desc\":\"\"},{\"name\":\"used\",\"value\":\"9101712\",\"category\":\"nodejs\",\"group\":\"HeapMemoryUsage\",\"timestamp\":1459331294985,\"unit\":\"Bytes\",\"desc\":\"\"},{\"name\":\"committed\",\"value\":\"14620256\",\"category\":\"nodejs\",\"group\":\"HeapMemoryUsage\",\"timestamp\":1459331294985,\"unit\":\"Bytes\",\"desc\":\"\"},{\"name\":\"max\",\"value\":\"1501560832\",\"category\":\"nodejs\",\"group\":\"HeapMemoryUsage\",\"timestamp\":1459331294985,\"unit\":\"Bytes\",\"desc\":\"\"},{\"name\":\"throughput\",\"value\":\"0\",\"category\":\"nodejs\",\"group\":\"Web\",\"timestamp\":1459331294985,\"unit\":\"\",\"desc\":\"\"},{\"name\":\"responseTime\",\"value\":\"0\",\"category\":\"nodejs\",\"group\":\"Web\",\"timestamp\":1459331294985,\"unit\":\"ms\",\"desc\":\"\"},{\"name\":\"eventloopLatency\",\"value\":\"0.07673447083333336\",\"category\":\"nodejs\",\"group\":\"Web\",\"timestamp\":1459331294985,\"unit\":\"ms\",\"desc\":\"\"}],\"stored\":false}],\"timestamp\":1459331295019,\"memQuota\":1024,\"type\":\"AppInstanceMetrics\",\"new\":true}";
		return RestApiResponseHandler.getResponseOk(metricStr);

	}

}

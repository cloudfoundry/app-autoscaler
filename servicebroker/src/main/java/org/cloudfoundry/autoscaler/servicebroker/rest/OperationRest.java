package org.cloudfoundry.autoscaler.servicebroker.rest;

import java.util.List;
import java.util.Map;
import java.util.Map.Entry;

import javax.servlet.http.HttpServletRequest;
import javax.ws.rs.GET;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.Context;
import javax.ws.rs.core.Response;

import org.apache.log4j.Level;
import org.apache.log4j.LogManager;
import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.servicebroker.mgr.ConfigManager;
import org.cloudfoundry.autoscaler.servicebroker.mgr.ScalingServiceMgr;
import org.json.JSONException;
import org.json.JSONObject;

@Path("/public")
public class OperationRest {

	@GET
	@Path("info")
	public Response buildInfo(@Context final HttpServletRequest httpServletRequest) {
		JSONObject response = new JSONObject();
		String version = ConfigManager.get("buildVersion");
		response.put("version", version);
		return Response.ok().entity(response.toString()).build();
	}

	@GET
	@Path("workloadstats")
	public Response workloadStats(@Context final HttpServletRequest httpServletRequest) {
	
		Map<String, Map<String, List<String>>> workloadStatsMap = ScalingServiceMgr.getInstance().getWorkloadStats();
		
		String htmlTitle = "<html><body><title>WorkLoad Stats</title>";
		StringBuilder htmlParagraph = new StringBuilder();
		
		StringBuilder htmlTable = new StringBuilder();
		htmlTable.append("<table width = '100%' border='1'><tr><th width = '30%'>Service URL</th><th width = '35%'>Service Id</th><th>App Id</th></tr>");

		for (Entry<String, Map<String, List<String>>> serverURLEntry : workloadStatsMap.entrySet() ){
			String serverUrl = serverURLEntry.getKey();
			Map<String, List<String>> serviceBindingMap = serverURLEntry.getValue();
			
			int serviceCount = serviceBindingMap.size();
			int appCount = 0;
			
			if (serviceCount == 0){
				//remove <td>
				htmlTable.append("<tr><td rowspan=1'>" + serverUrl + "</td>");
			} else{ 
			
				htmlTable.append("<tr><td rowspan='" + serviceCount + "'>" + serverUrl + "</td><td>");
				for (Entry<String, List<String>> serviceEntry:  serviceBindingMap.entrySet() ){
					String serviceId = serviceEntry.getKey();
					List<String> appIds = serviceEntry.getValue();
					htmlTable.append(serviceId).append("</td><td>");
					if (appIds != null) {
						appCount += appIds.size();
						for(String appId: appIds) {
							htmlTable.append(appId).append("<br/>");
						}
					}
					htmlTable.append("</td></tr><tr><td>");
				}

				//remove the last <tr><td>
				htmlTable.delete(htmlTable.length() - 8, htmlTable.length());
			
			}
			htmlParagraph.append("<p>" + serverUrl + ": " + serviceCount + " services provisioned, " + appCount + " apps bound; </p>");
		}
		
		htmlTable.append("</table></body></html>");
		
		String htmlEntity = htmlTitle + htmlParagraph.toString() + htmlTable.toString();
		return Response.ok(htmlEntity).build();
	}

}

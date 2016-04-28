package org.cloudfoundry.autoscaler.metric.rest;

import java.util.Set;

import javax.ws.rs.GET;
import javax.ws.rs.PUT;
import javax.ws.rs.Path;
import javax.ws.rs.PathParam;
import javax.ws.rs.QueryParam;
import javax.ws.rs.core.Response;

import org.apache.log4j.Level;
import org.apache.log4j.LogManager;
import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.cloudfoundry.autoscaler.common.util.RestApiResponseHandler;
import org.cloudfoundry.autoscaler.data.couchdb.dao.base.CouchDBConnectionProfile;
import org.json.JSONObject;

@Path("/operation")
public class OperationREST {

	private static final Logger logger = Logger.getLogger(OperationREST.class);

	@PUT
	@Path("/log/{level}")
	public Response setLogLevel(@PathParam("level") final String level,
			@QueryParam("package") final String packageName) {

		String targetLoggerName = "org.cloudfoundry.autoscaler";

		if (packageName != null && !packageName.isEmpty())
			targetLoggerName = packageName;

		Logger targetLogger = LogManager.getLogger(targetLoggerName);

		if (level.equalsIgnoreCase("INFO"))
			targetLogger.setLevel(Level.INFO);
		else if (level.equalsIgnoreCase("DEBUG"))
			targetLogger.setLevel(Level.DEBUG);
		else if (level.equalsIgnoreCase("ERROR"))
			targetLogger.setLevel(Level.ERROR);
		else if (level.equalsIgnoreCase("WARN"))
			targetLogger.setLevel(Level.WARN);
		else if (level.equalsIgnoreCase("FATAL"))
			targetLogger.setLevel(Level.FATAL);
		else if (level.equalsIgnoreCase("TRACE"))
			targetLogger.setLevel(Level.TRACE);
		else if (level.equalsIgnoreCase("OFF"))
			targetLogger.setLevel(Level.OFF);
		else if (level.equalsIgnoreCase("ALL"))
			targetLogger.setLevel(Level.ALL);

		logger.info("Log level " + level.toUpperCase() + " is set");
		return RestApiResponseHandler.getResponseOk();

	}

	@GET
	@Path("/info")
	public Response info() {

		JSONObject response = new JSONObject();
		String version = ConfigManager.get("build.version");
		response.put("version", version);
		return RestApiResponseHandler.getResponseOk(response);
	}

	@GET
	@Path("/connection")
	public Response connectoins() {
		int count = CouchDBConnectionProfile.getInstance().getConnectionCount();

		Set<String> connections = CouchDBConnectionProfile.getInstance().getConntionUUID();

		StringBuilder response = new StringBuilder().append(count).append("\n");
		for (String str : connections) {
			response.append(str).append("\n");
		}
		return RestApiResponseHandler.getResponseOk(response.toString());
	}

}

package org.cloudfoundry.autoscaler.servicebroker.mgr;

import java.io.IOException;

import org.apache.commons.io.IOUtils;
import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.common.util.ConfigManager;
import org.json.JSONObject;

public class Catalog {

	private static final Logger logger = Logger.getLogger(Catalog.class);

	public static JSONObject getCatalogJSON() {
		String CATALOG_FILE = "/" + ConfigManager.get("catalogFile", "catalog.json");
		try {
			return new JSONObject(IOUtils.toString(Catalog.class.getResourceAsStream(CATALOG_FILE), "UTF-8"));
		} catch (IOException e) {
			logger.error("Invalid catalog file");
		}
		return null;
	}
}

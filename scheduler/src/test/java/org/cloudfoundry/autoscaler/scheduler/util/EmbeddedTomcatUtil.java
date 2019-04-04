package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.File;
import java.io.IOException;

import javax.servlet.ServletException;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.apache.catalina.Context;
import org.apache.catalina.LifecycleException;
import org.apache.catalina.Service;
import org.apache.catalina.connector.Connector;
import org.apache.catalina.startup.Tomcat;
import org.apache.tomcat.util.http.fileupload.FileUtils;

public class EmbeddedTomcatUtil {

	private File baseDir = null;

	private File applicationDir;

	private Context appContext;

	private Tomcat tomcat = new Tomcat();

	public EmbeddedTomcatUtil() {
		baseDir = new File("tomcat");
		tomcat.setBaseDir(baseDir.getAbsolutePath());

		Connector httpsConnector = new Connector();
		setupSSLConfig(httpsConnector);

		Service service = tomcat.getService();
		service.addConnector(httpsConnector);

		applicationDir = new File(baseDir + "/webapps", "/ROOT");

		if (!applicationDir.exists()) {
			applicationDir.mkdirs();
		}
		tomcat.setPort(8090);
		tomcat.setSilent(false);

	}

	public void start() {
		try {
			tomcat.start();
			appContext = tomcat.addWebapp("/", applicationDir.getAbsolutePath());
		} catch (LifecycleException e) {
			throw new RuntimeException(e);
		}
	}

	public void stop() {
		try {
			tomcat.stop();
			tomcat.destroy();
			// Tomcat creates a work folder where the temporary files are stored
			FileUtils.deleteDirectory(baseDir);
		} catch (LifecycleException | IOException e) {
			throw new RuntimeException(e);
		}
	}

	public void setup(String appId, Long scheduleId, int statusCode, String message) throws ServletException {
		String url = "/v1/apps/" + appId + "/active_schedules/" + scheduleId;
		tomcat.addServlet(appContext.getPath(), appId, new ScalingEngineMock(statusCode, message));
		appContext.addServletMappingDecoded(url, appId);
	}

	public void setup(String appId, int statusCode, String message) {
		String url = "/v1/apps/" + appId + "/active_schedules/*";
		tomcat.addServlet(appContext.getPath(), appId, new ScalingEngineMock(statusCode, message));
		appContext.addServletMappingDecoded(url, appId);
	}

	private void setupSSLConfig(Connector httpsConnector) {
		httpsConnector.setPort(8091);
		httpsConnector.setSecure(true);
		httpsConnector.setScheme("https");
		httpsConnector.setAttribute("truststoreFile", "../src/test/resources/certs/test.truststore");
		httpsConnector.setAttribute("truststorePass", "123456");
		httpsConnector.setAttribute("keystoreFile", "../src/test/resources/certs/fake-scalingengine.p12");
		httpsConnector.setAttribute("keyAlias", "fake-scalingengine");
		httpsConnector.setAttribute("keystorePass", "123456");
		httpsConnector.setAttribute("clientAuth", "true");
		httpsConnector.setAttribute("sslProtocol", "TLSv1.2");
		httpsConnector.setAttribute("SSLEnabled", true);
	}

	static class ScalingEngineMock extends HttpServlet {

		private int returnStatus;
		private String returnMessage;

		ScalingEngineMock(int status, String returnMessage) {
			this.returnStatus = status;
			this.returnMessage = returnMessage;
		}

		@Override
		protected void doPut(HttpServletRequest request, HttpServletResponse response) throws IOException {

			response.setStatus(this.returnStatus);
			if (returnMessage != null && !returnMessage.isEmpty()) {
				response.getWriter().write(returnMessage);
				response.setContentType("application/json");
			}
		}

		@Override
		protected void doDelete(HttpServletRequest request, HttpServletResponse response) throws IOException {

			response.setStatus(this.returnStatus);
			if (returnMessage != null && !returnMessage.isEmpty()) {
				response.getWriter().write(returnMessage);
				response.setContentType("application/json");
			}
		}

	}

}

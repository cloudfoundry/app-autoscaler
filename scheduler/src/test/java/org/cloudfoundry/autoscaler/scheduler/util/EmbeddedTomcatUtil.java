package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.File;
import java.io.IOException;

import javax.servlet.ServletException;
import javax.servlet.http.HttpServlet;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;

import org.apache.catalina.Context;
import org.apache.catalina.LifecycleException;
import org.apache.catalina.startup.Tomcat;
import org.apache.tomcat.util.http.fileupload.FileUtils;

public class EmbeddedTomcatUtil {
	File applicationDir;
	Context appContext;

	public EmbeddedTomcatUtil() {
		File baseDir = new File("tomcat");
		tomcat.setBaseDir(baseDir.getAbsolutePath());

		applicationDir = new File(baseDir + "/webapps", "/ROOT");

		if (!applicationDir.exists()) {
			applicationDir.mkdirs();
		}
		tomcat.setPort(8090);
		tomcat.setSilent(false);
	}

	private Tomcat tomcat = new Tomcat();

	public void start() {
		try {
			tomcat.start();
			appContext = tomcat.addWebapp("/", applicationDir.getAbsolutePath());
		} catch (LifecycleException e) {
			throw new RuntimeException(e);
		} catch (ServletException e) {
			throw new RuntimeException(e);
		}
	}

	public void stop() {
		try {
			tomcat.stop();
			tomcat.destroy();
			// Tomcat creates a work folder where the temporary files are stored
			FileUtils.deleteDirectory(new File("work"));
		} catch (LifecycleException e) {
			throw new RuntimeException(e);
		}

		catch (IOException e) {
			throw new RuntimeException(e);
		}
	}

	public void setup(String appId, Long scheduleId, int statusCode, String message) throws ServletException {
		String url = "/v1/apps/" + appId + "/active_schedules/" + scheduleId;
		Tomcat.addServlet(appContext, appId, new ScalingEngineMock(statusCode, message));
		appContext.addServletMapping(url, appId);

	}

	static class ScalingEngineMock extends HttpServlet {

		private int returnStatus;
		private String returnMessage;

		public ScalingEngineMock(int status, String returnMessage) {
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

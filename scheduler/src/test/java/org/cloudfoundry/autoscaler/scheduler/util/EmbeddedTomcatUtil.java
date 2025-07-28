package org.cloudfoundry.autoscaler.scheduler.util;

import jakarta.servlet.ServletException;
import jakarta.servlet.http.HttpServlet;
import jakarta.servlet.http.HttpServletRequest;
import jakarta.servlet.http.HttpServletResponse;
import java.io.File;
import java.io.IOException;
import org.apache.catalina.Context;
import org.apache.catalina.LifecycleException;
import org.apache.catalina.Service;
import org.apache.catalina.connector.Connector;
import org.apache.catalina.startup.Tomcat;
import org.apache.tomcat.util.http.fileupload.FileUtils;
import org.apache.tomcat.util.net.SSLHostConfig;
import org.apache.tomcat.util.net.SSLHostConfigCertificate;

public class EmbeddedTomcatUtil {

  private File baseDir = null;

  private File applicationDir;

  private Context appContext;

  private Tomcat tomcat = new Tomcat();

  public EmbeddedTomcatUtil() {
    baseDir = new File("tomcat");
    tomcat.setBaseDir(baseDir.getAbsolutePath());

    Connector httpsConnector = new Connector();
    setupSslConfig(httpsConnector);

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

  public void addScalingEngineMockForAppAndScheduleId(
      String appId, Long scheduleId, int statusCode, String message) throws ServletException {
    String url = "/v1/apps/" + appId + "/active_schedules/" + scheduleId;
    tomcat.addServlet(appContext.getPath(), appId, new ScalingEngineMock(statusCode, message));
    appContext.addServletMappingDecoded(url, appId);
  }

  public void addScalingEngineMockForAppId(String appId, int statusCode, String message) {
    String url = "/v1/apps/" + appId + "/active_schedules/*";
    tomcat.addServlet(appContext.getPath(), appId, new ScalingEngineMock(statusCode, message));
    appContext.addServletMappingDecoded(url, appId);
  }

  private void setupSslConfig(Connector httpsConnector) {
    httpsConnector.setPort(8091);
    httpsConnector.setSecure(true);
    httpsConnector.setScheme("https");

    httpsConnector.setProperty("clientAuth", "true");
    httpsConnector.setProperty("sslProtocol", "TLSv1.2");
    httpsConnector.setProperty("SSLEnabled", "true");

    SSLHostConfig sslConfig = new SSLHostConfig();

    SSLHostConfigCertificate certConfig =
        new SSLHostConfigCertificate(sslConfig, SSLHostConfigCertificate.Type.RSA);
    certConfig.setCertificateKeystoreFile("../src/test/resources/certs/fake-scalingengine.p12");
    certConfig.setCertificateKeystorePassword("123456");
    certConfig.setCertificateKeyAlias("fake-scalingengine");

    sslConfig.addCertificate(certConfig);

    sslConfig.setTruststoreFile("../src/test/resources/certs/test.truststore");
    sslConfig.setTruststorePassword("123456");

    httpsConnector.addSslHostConfig(sslConfig);
  }

  static class ScalingEngineMock extends HttpServlet {

    private int returnStatus;
    private String returnMessage;

    ScalingEngineMock(int status, String returnMessage) {
      this.returnStatus = status;
      this.returnMessage = returnMessage;
    }

    @Override
    protected void doPut(HttpServletRequest request, HttpServletResponse response)
        throws IOException {

      response.setStatus(this.returnStatus);
      if (returnMessage != null && !returnMessage.isEmpty()) {
        response.getWriter().write(returnMessage);
        response.setContentType("application/json");
      }
    }

    @Override
    protected void doDelete(HttpServletRequest request, HttpServletResponse response)
        throws IOException {

      response.setStatus(this.returnStatus);
      if (returnMessage != null && !returnMessage.isEmpty()) {
        response.getWriter().write(returnMessage);
        response.setContentType("application/json");
      }
    }
  }
}

package org.cloudfoundry.autoscaler.scheduler.conf;

import com.sun.net.httpserver.BasicAuthenticator;
import com.sun.net.httpserver.HttpExchange;
import java.util.Map;

public class MetricsBasicAuthentication extends BasicAuthenticator {

  private final Map<String, String> users;

  public MetricsBasicAuthentication(String realm, Map<String, String> users) {
    super(realm);
    this.users = users;
  }

  @Override
  public Result authenticate(HttpExchange exch) {
    Result result = super.authenticate(exch);
    if (result instanceof Success) {
      String requestMethod = exch.getRequestMethod();
      // lock all requests down to GET
      if (!requestMethod.equals("GET")) {
        return new Failure(401);
      }
      return result;
    }
    return result;
  }

  @Override
  public boolean checkCredentials(String user, String pwd) {
    String passHash = users.get(user);
    return passHash != null && passHash.equals(pwd);
  }
}

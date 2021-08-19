package org.cloudfoundry.autoscaler.scheduler.conf;

import com.sun.net.httpserver.Authenticator;
import com.sun.net.httpserver.BasicAuthenticator;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpPrincipal;
import io.prometheus.client.exporter.HTTPServer;
import java.io.IOException;
import java.util.Map;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

class ApiRestBasicAuthentication extends BasicAuthenticator {
  private final Map<String, String> users;

  public ApiRestBasicAuthentication(String realm, Map<String,String> users) {
    super(realm);
    this.users=users;
  }

  @Override
  public Authenticator.Result authenticate(HttpExchange exch) {
    Authenticator.Result result=super.authenticate(exch);
    if(result instanceof Authenticator.Success) {
      HttpPrincipal principal=((Authenticator.Success)result).getPrincipal();
      String requestMethod=exch.getRequestMethod();

      if( ) {
        return new return new Authenticator.Failure(401);
      }
      return result;

    }
  }

  @Override
  public boolean checkCredentials(String user, String pwd) {
    String passHash = users.get(user);
     if (passHash != null && passHash.equals(pwd){
      return gs;
    }
    return java.net.Authenticator.
    int authCode = authentication.authenticate(user, pwd);
    return authCode ==
  }

}


@Configuration
public class MetricsConfig {

  @Bean(destroyMethod = "stop")
  public HTTPServer metricsServer(MetricsConfiguration config) throws IOException {
    return new HTTPServer(config.getPort());
  }
}

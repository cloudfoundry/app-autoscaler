package org.cloudfoundry.autoscaler.scheduler.conf;

import io.prometheus.client.vertx.MetricsHandler;
import io.vertx.core.AsyncResult;
import io.vertx.core.Future;
import io.vertx.core.Handler;
import io.vertx.core.Vertx;
import io.vertx.core.json.JsonObject;
import io.vertx.ext.auth.AuthProvider;
import io.vertx.ext.auth.User;
import io.vertx.ext.web.Router;
import io.vertx.ext.web.handler.BasicAuthHandler;
import org.springframework.context.annotation.Bean;
import org.springframework.context.annotation.Configuration;

@Configuration
public class MetricsConfig {

  @Bean(destroyMethod = "close")
  Vertx metricsServer(MetricsConfiguration config) {
    final Vertx vertx = Vertx.vertx();
    final Router router = Router.router(vertx);
    if (config.isBasicAuthEnabled()) {
      router.route("/*").handler(BasicAuthHandler.create(getAuthProvider(config)));
    }
    router.route("/metrics").handler(new MetricsHandler());
    vertx.createHttpServer().requestHandler(router).listen(config.getPort());
    return vertx;
  }

  private AuthProvider getAuthProvider(MetricsConfiguration config) {
    return (JsonObject authInfo, Handler<AsyncResult<User>> resultHandler) -> {
      String password = authInfo.getString("password");
      String username = authInfo.getString("username");
      if (config.getUsername().equals(username) && config.getPassword().equals(password)) {
        resultHandler.handle(Future.succeededFuture(null));
      } else {
        resultHandler.handle(Future.failedFuture("Not authorised"));
      }
    };
  }
}

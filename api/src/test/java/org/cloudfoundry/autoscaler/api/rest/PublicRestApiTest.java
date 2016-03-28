package org.cloudfoundry.autoscaler.api.rest;

import static org.junit.Assert.assertEquals;

import com.google.inject.Guice;
import com.google.inject.Injector;
import com.google.inject.Provider;
import com.google.inject.servlet.RequestScoped;
import com.google.inject.servlet.ServletModule;
import com.google.inject.servlet.ServletScopes;
import com.sun.jersey.api.client.Client;
import com.sun.jersey.api.client.ClientResponse;
import com.sun.jersey.api.client.WebResource;
import com.sun.jersey.api.client.config.DefaultClientConfig;
import com.sun.jersey.api.container.grizzly2.GrizzlyServerFactory;
import com.sun.jersey.api.core.PackagesResourceConfig;
import com.sun.jersey.api.core.ResourceConfig;
import com.sun.jersey.core.spi.component.ioc.IoCComponentProviderFactory;
import com.sun.jersey.guice.JerseyServletModule;
import com.sun.jersey.guice.spi.container.GuiceComponentProviderFactory;
import com.sun.jersey.spi.container.servlet.ServletContainer;
import com.sun.jersey.guice.spi.container.servlet.GuiceContainer;

import org.glassfish.grizzly.http.server.HttpServer;
import org.glassfish.grizzly.servlet.WebappContext;
import org.junit.After;
import org.junit.Before;
import org.junit.Test;

import javax.servlet.ServletRegistration;
import javax.servlet.ServletRequest;
import javax.servlet.ServletResponse;
import javax.servlet.http.HttpServletRequest;
import javax.servlet.http.HttpServletResponse;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.UriBuilder;
import java.io.IOException;
import java.net.URI;

public class PublicRestApiTest {
    static final URI BASE_URI = getBaseURI();
    HttpServer server;
    static final String app_id = "d76e90be-15fb-43ac-b9d7-78f81918d77a";

    private static URI getBaseURI() {
        return UriBuilder.fromUri("http://localhost/").port(9998).build();
    }

/*    private static <T> Provider<T> unusableProvider(final Class<T> type) {
        return new Provider<T>() {
            @Override
            public T get() {
                throw new IllegalStateException("Unexpected call to provider of " + type.getSimpleName());
            }
        };
    }*/
    @Before
    public void startServer() throws IOException {
        System.out.println("Starting grizzly...");

        Injector injector = Guice.createInjector(new JerseyServletModule() {
            @Override
            protected void configureServlets() {
/*              bindScope(RequestScoped.class, ServletScopes.REQUEST);

                bind(ServletRequest.class).toProvider(unusableProvider(ServletRequest.class)).in(RequestScoped.class);
                bind(HttpServletRequest.class).toProvider(unusableProvider(HttpServletRequest.class)).in(RequestScoped.class);
                bind(ServletResponse.class).toProvider(unusableProvider(ServletResponse.class)).in(RequestScoped.class);
                bind(HttpServletResponse.class).toProvider(unusableProvider(HttpServletResponse.class)).in(RequestScoped.class);*/	
            	
               bind(PublicRestApi.class);
               serve("/*").with(GuiceContainer.class);
            }
        });
        System.out.println("Trying to find provider");
        ResourceConfig rc = new PackagesResourceConfig("org.cloudfoundry.autoscaler.api.rest");
        IoCComponentProviderFactory ioc = new GuiceComponentProviderFactory(rc, injector);
        server = GrizzlyServerFactory.createHttpServer(BASE_URI, rc, ioc);
        
        /*
        HttpServer server = GrizzlyServerFactory.createHttpServer(BASE_URI, rc);
        System.out.println("!!!!!!!!!!!!!!  Prepare the Servlet");
        WebappContext context = new WebappContext("GrizzlyContext", "/");
        ServletRegistration registration = context.addServlet(
                ServletContainer.class.getName(), ServletContainer.class);
        registration.setInitParameter(ServletContainer.RESOURCE_CONFIG_CLASS,
                PackagesResourceConfig.class.getName());
        registration.setInitParameter(PackagesResourceConfig.PROPERTY_PACKAGES, "org.cloudfoundry.autoscaler.api.rest");
        registration.addMapping("/*");
        context.deploy(server);
        System.out.println("!!!!!!!!!!!!!!!!!! Prepare the Servlet Done");
        server.start();
        */
        //TODO comment not correct
        System.out.println(String.format("Jersey app started with WADL available at "
                + "%srest/application.wadl\nTry out %sngdemo\nHit enter to stop it...",
                BASE_URI, BASE_URI));
    }

    @After
    public void stopServer() {
        server.stop();
    }
    
    //@Test
    public void testWithNoToken() throws IOException {
        Client client = Client.create(new DefaultClientConfig());
        WebResource service = client.resource(getBaseURI());
        System.out.println("Prepare the Rest Call ");
        ClientResponse resp = service.path("apps").path(app_id).path("policy")
                .accept(MediaType.APPLICATION_JSON)
                .get(ClientResponse.class);
        System.out.println("Got stuff: " + resp);
        String text = resp.getEntity(String.class);
        System.out.println("Response body:" + text);

        assertEquals(401, resp.getStatus());
    }
}

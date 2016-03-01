package org.cloudfoundry.autoscaler.servicebroker.restclient;

import javax.ws.rs.core.Cookie;
import javax.ws.rs.core.MediaType;
public interface IRestClient {

	public IRestResourceBuilder resource(String url);

	public interface IRestResourceBuilder {

		public IRestResourceBuilder header(String name, Object value);
		public IRestResourceBuilder accept(MediaType mediaType);
		public IRestResourceBuilder accept(String mediaType);
		public IRestResourceBuilder type(MediaType mediaType);
		public IRestResourceBuilder type(String mediaType);
		public IRestResourceBuilder cookie(Cookie cookie);

		public <T> T get(Class<T> c);
		public <T> T post(Class<T> c);
		public <T> T post(Class<T> c,Object requestEntity);
		public <T> T put(Class<T> c);
		public <T> T put(Class<T> c, Object requestEntity);
		public <T> T delete(Class<T> c);
	}
}

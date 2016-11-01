package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.DataOutputStream;
import java.io.IOException;
import java.net.HttpURLConnection;
import java.net.URL;

import org.apache.logging.log4j.LogManager;
import org.apache.logging.log4j.Logger;
import org.cloudfoundry.autoscaler.scheduler.entity.ActiveScheduleEntity;
import org.springframework.http.MediaType;
import org.springframework.stereotype.Component;

import com.fasterxml.jackson.databind.ObjectMapper;

@Component
public class ScalingEngineUtil {
	private Logger logger = LogManager.getLogger(this.getClass());

	private HttpURLConnection httpURLConnection = null;
	private DataOutputStream dataOutputStream = null;

	public HttpURLConnection getConnection(URL url, ActiveScheduleEntity activeScheduleEntity) throws IOException {

		httpURLConnection = (HttpURLConnection) url.openConnection();
		httpURLConnection.setRequestMethod("PUT");
		httpURLConnection.setRequestProperty("Content-Type", MediaType.APPLICATION_JSON_VALUE);
		httpURLConnection.setDoInput(true);
		httpURLConnection.setDoOutput(true);

		ObjectMapper objectMapper = new ObjectMapper();
		String content = objectMapper.writeValueAsString(activeScheduleEntity);

		dataOutputStream = new DataOutputStream(httpURLConnection.getOutputStream());
		dataOutputStream.write(content.getBytes());

		return httpURLConnection;
	}

	public void close() {
		if (dataOutputStream != null) {
			try {
				dataOutputStream.flush();
				dataOutputStream.close();
			} catch (IOException ie) {
				logger.error(ie.getMessage(), ie);
			}
		}
		if (httpURLConnection != null) {
			httpURLConnection.disconnect();
		}
	}

}

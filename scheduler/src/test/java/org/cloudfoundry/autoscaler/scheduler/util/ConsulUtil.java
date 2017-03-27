package org.cloudfoundry.autoscaler.scheduler.util;

import java.io.File;
import java.io.IOException;

import org.apache.tomcat.util.http.fileupload.FileUtils;

public class ConsulUtil {

	private static final String DATA_DIR = "/tmp/consul_agent";
	private Process process;
	private File dataDir;

	public void start() throws IOException {
		dataDir = new File(DATA_DIR);

		if (!dataDir.isDirectory()) {
			dataDir.mkdirs();
		}

		String[] cmds = new String[] { "../bin/consul", "agent", "-config-file", "src/test/resources/consul.config" };
		ProcessBuilder processBuilder = new ProcessBuilder(cmds);
		process = processBuilder.start();
	}

	public void stop() throws IOException, InterruptedException {
		if (process != null) {
			process.destroy();
			process.waitFor();
		}
		if (dataDir != null) {
			FileUtils.deleteDirectory(dataDir);
		}
	}
}

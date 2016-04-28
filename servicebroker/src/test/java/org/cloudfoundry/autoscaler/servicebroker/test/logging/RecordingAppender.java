package org.cloudfoundry.autoscaler.servicebroker.test.logging;

import java.util.ArrayList;
import java.util.List;

import org.apache.log4j.AppenderSkeleton;
import org.apache.log4j.PatternLayout;
import org.apache.log4j.spi.LoggingEvent;

public class RecordingAppender extends AppenderSkeleton {
	private static List<String> messages = new ArrayList<String>();
	private static RecordingAppender appender = new RecordingAppender();

	private RecordingAppender() {
		super();
	}

	public static RecordingAppender appender(PatternLayout patternLayout) {
		appender.setLayout(patternLayout);
		appender.clear();
		return appender;
	}

	protected void append(LoggingEvent event) {
		messages.add(layout.format(event));
	}

	public void close() {
	}

	public boolean requiresLayout() {
		return true;
	}

	public static String[] messages() {
		return (String[]) messages.toArray(new String[messages.size()]);
	}

	private void clear() {
		messages.clear();
	}
}

package org.cloudfoundry.autoscaler.test.testcase.base;

import org.apache.log4j.ConsoleAppender;
import org.apache.log4j.Level;
import org.apache.log4j.Logger;
import org.apache.log4j.PatternLayout;
import org.cloudfoundry.autoscaler.rest.mock.couchdb.CouchDBDocumentManager;
import org.cloudfoundry.autoscaler.test.logging.RecordingAppender;

import com.sun.jersey.test.framework.JerseyTest;

public class JerseyTestBase extends JerseyTest {

	public JerseyTestBase() throws Exception {
		super("org.cloudfoundry.autoscaler.rest");
		configureLog();
	}
	public JerseyTestBase(String... packages) throws Exception {
		super(packages);
		configureLog();
	}

	@Override
	public void tearDown() throws Exception {
		super.tearDown();
		CouchDBDocumentManager.getInstance().initDocuments();
	}

	protected boolean logContains(String expected) {
		String actual[] = RecordingAppender.messages();
		for (String log : actual) {
		if (log.indexOf(expected) >= 0)
		return true;
		}
		return false;
	}

	private void configureLog() {
		Logger rootLogger = Logger.getRootLogger();
		rootLogger.removeAllAppenders();
		rootLogger.setLevel(Level.INFO);
		rootLogger.addAppender(new ConsoleAppender(new PatternLayout(
		"%d [%t] %-5p %c{1} - %m%n")));
		rootLogger.addAppender(RecordingAppender.appender(new PatternLayout("%-5p - %m%n")));
	}

}


package org.cloudfoundry.autoscaler.servicebroker.test.util;

import java.util.Random;

/**
 *
 */
public class TestConstants {

    public final static int RANDOM_TEST_ID = Math.abs(new Random().nextInt());
    public final static String TEST_APPLICATION_ID = "testAppID" + RANDOM_TEST_ID;
    public final static String TEST_SERVICE_ID = "testServiceID" + RANDOM_TEST_ID;
    public final static String TEST_BINDING_ID = "testBindingID" + RANDOM_TEST_ID;

    public final static String TEST_ORG_ID = "testOrgID" + RANDOM_TEST_ID;
    public final static String TEST_SPACE_ID = "testSpaceID" + RANDOM_TEST_ID;
    public final static String TEST_SERVER_URL = "testServerURL" + RANDOM_TEST_ID;

    public final static String MOCK_SERVER_URL = "http://localhost:8030/AutoScalingServerMock";
    public final static String MOCK_SERVER_TEST_SUCCESS_URL = "/resources/test/success";
    public final static String MOCK_SERVER_TEST_FAILURE_URL = "/resources/test/failure";

}

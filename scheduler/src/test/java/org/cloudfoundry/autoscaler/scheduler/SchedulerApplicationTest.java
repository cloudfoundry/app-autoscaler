package org.cloudfoundry.autoscaler.scheduler;

import org.junit.After;
import org.junit.Before;
import org.junit.Test;
import org.junit.runner.RunWith;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.junit4.SpringRunner;

import java.security.Permission;

import static org.junit.Assert.assertEquals;
import static org.junit.Assert.fail;

@RunWith(SpringRunner.class)
@SpringBootTest
public class SchedulerApplicationTest {
    protected static class ExitException extends SecurityException {
        private static final long serialVersionUID = -1982617086752946683L;
        private final int status;

        private ExitException(int status) {
            this.status = status;
        }

        private int getStatus(){
            return status;
        }
    }

    private static class NoExitSecurityManager extends SecurityManager {
        @Override
        public void checkPermission(Permission perm) {}

        @Override
        public void checkPermission(Permission perm, Object context) {}

        @Override
        public void checkExit(int status) {
            super.checkExit(status);
            throw new ExitException(status);
        }
    }

    private SecurityManager securityManager;

    @Before
    public void before() {
        securityManager = System.getSecurityManager();
        System.setSecurityManager(new NoExitSecurityManager());
    }

    @After
    public void tearDown() {
        System.setSecurityManager(securityManager);
    }

    @Test
    public void testApplicationExitsWhenDBUnreachable()
    {
        try {
            SchedulerApplication.main(new String[]{
                    "--spring.cloud.consul.enabled=false",
                    "--spring.datasource.url=jdbc:postgresql://127.0.0.1/autoscaler1"
            });

            fail("Expected exit");
        } catch (ExitException e) {
            int status = e.getStatus();
            assertEquals(1, status);
        }
    }
}
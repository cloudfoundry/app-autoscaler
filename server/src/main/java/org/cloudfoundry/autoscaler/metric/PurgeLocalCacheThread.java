package org.cloudfoundry.autoscaler.metric;

import org.apache.log4j.Logger;
import org.cloudfoundry.autoscaler.application.ApplicationManagerImpl;
import org.cloudfoundry.autoscaler.policy.PolicyManagerImpl;

public class PurgeLocalCacheThread implements Runnable {
    private static final Logger logger = Logger.getLogger(PurgeLocalCacheThread.class);
  
    public PurgeLocalCacheThread() {
    }

    @Override
    public void run() {
    	ApplicationManagerImpl.getInstance().invalidateCache();
    	PolicyManagerImpl.getInstance().invalidateCache();
    }
}

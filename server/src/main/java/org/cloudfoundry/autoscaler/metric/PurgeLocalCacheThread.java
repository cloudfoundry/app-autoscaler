package org.cloudfoundry.autoscaler.metric;

import org.cloudfoundry.autoscaler.application.ApplicationManagerImpl;
import org.cloudfoundry.autoscaler.policy.PolicyManagerImpl;

public class PurgeLocalCacheThread implements Runnable {
    public PurgeLocalCacheThread() {
    }

    @Override
    public void run() {
    	ApplicationManagerImpl.getInstance().invalidateCache();
    	PolicyManagerImpl.getInstance().invalidateCache();
    }
}

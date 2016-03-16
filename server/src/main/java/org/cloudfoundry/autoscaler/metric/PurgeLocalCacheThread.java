package org.cloudfoundry.autoscaler.metric;

import org.cloudfoundry.autoscaler.manager.ApplicationManagerImpl;
import org.cloudfoundry.autoscaler.manager.PolicyManagerImpl;

public class PurgeLocalCacheThread implements Runnable {
    public PurgeLocalCacheThread() {
    }

    @Override
    public void run() {
    	ApplicationManagerImpl.getInstance().invalidateCache();
    	PolicyManagerImpl.getInstance().invalidateCache();
    }
}

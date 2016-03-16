package org.cloudfoundry.autoscaler.data.couchdb.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.data.couchdb.document.ScalingHistory;

public interface ScalingHistoryDAO extends CommonDAO {

	public List<ScalingHistory> findAll();

	public List<ScalingHistory> findByScalingTime(String appId, long startTime, long endTime);

}

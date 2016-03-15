package org.cloudfoundry.autoscaler.cloudservice.couchdb.data.dao;

import java.util.List;

import org.cloudfoundry.autoscaler.cloudservice.couchdb.data.document.ScalingHistory;

public interface ScalingHistoryDAO extends CommonDAO {

	public List<ScalingHistory> findAll();

	public List<ScalingHistory> findByScalingTime(String appId, long startTime, long endTime);

}

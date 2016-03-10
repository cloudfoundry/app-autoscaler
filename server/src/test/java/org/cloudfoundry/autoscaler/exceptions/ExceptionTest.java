package org.cloudfoundry.autoscaler.exceptions;

import static org.junit.Assert.assertEquals;

import org.junit.Test;

public class ExceptionTest {
	
	@Test
	public void ExceptionsTest(){
		String testExceptionMsg = "This is a test exception"; 
		String appId = "appId";
		String accountId = "accountId";
		int errorCode = 0;
		String orgId = "orgId";
		String spaceId = "spaceId";
		String policyId = "policyId";
		
		try {
			throw new AppNotFoundException(appId);
		} catch (AppNotFoundException e) {
			assertEquals(appId, e.getAppId());
		}
		try {
			throw new AppNotFoundException(appId, testExceptionMsg);
		} catch (AppNotFoundException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			assertEquals(appId, e.getAppId());
		}
		
		try {
			throw new AuthorizationException(accountId);
		} catch (AuthorizationException e) {
			assertEquals(accountId, e.getAccountId());
		}
		try {
			throw new AuthorizationException(accountId, testExceptionMsg);
		} catch (AuthorizationException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			assertEquals(accountId, e.getAccountId());
		}
		
		try {
			throw new DataStoreException(testExceptionMsg);
		} catch (DataStoreException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			
		}
		
		try {
			throw new MetricNotSupportedException(appId);
		} catch (MetricNotSupportedException e) {
			assertEquals(appId, e.getAppId());
		}
		try {
			throw new MetricNotSupportedException(appId, testExceptionMsg);
		} catch (MetricNotSupportedException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			assertEquals(appId, e.getAppId());
		}
		
		try {
			throw new MonitorServiceException(testExceptionMsg);
		} catch (MonitorServiceException e) {
			assertEquals(testExceptionMsg, e.getMessage());
		}
		try {
			throw new MonitorServiceException(errorCode, testExceptionMsg);
		} catch (MonitorServiceException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			assertEquals(errorCode, e.getErrorCode());
		}
		
		try {
			throw new NoAttachedPolicyException(testExceptionMsg);
		} catch (NoAttachedPolicyException e) {
			assertEquals(testExceptionMsg, e.getMessage());
		}
		
		try {
			throw new NoMonitorServiceBoundException(testExceptionMsg);
		} catch (NoMonitorServiceBoundException e) {
			assertEquals(testExceptionMsg, e.getMessage());
		}
		
		try {
			throw new ObjectMapperException(testExceptionMsg);
		} catch (ObjectMapperException e) {
			assertEquals(testExceptionMsg, e.getMessage());
		}
		
		try {
			throw new OrgNotFoundException(orgId);
		} catch (OrgNotFoundException e) {
			assertEquals(orgId, e.getSpaceId());
		}
		try {
			throw new OrgNotFoundException(orgId, testExceptionMsg);
		} catch (OrgNotFoundException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			assertEquals(orgId, e.getSpaceId());
		}
		
		try {
			throw new PolicyNotFoundException(policyId);
		} catch (PolicyNotFoundException e) {
			assertEquals(policyId, e.getPolicyId());
		}
		try {
			throw new PolicyNotFoundException(policyId, testExceptionMsg);
		} catch (PolicyNotFoundException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			assertEquals(policyId, e.getPolicyId());
		}
		
		try {
			throw new SpaceNotFoundException(spaceId);
		} catch (SpaceNotFoundException e) {
			assertEquals(spaceId, e.getSpaceId());
		}
		try {
			throw new SpaceNotFoundException(spaceId, testExceptionMsg);
		} catch (SpaceNotFoundException e) {
			assertEquals(testExceptionMsg, e.getMessage());
			assertEquals(spaceId, e.getSpaceId());
		}
		
		try {
			throw new TriggerNotSubscribedException(appId);
		} catch (TriggerNotSubscribedException e) {
			assertEquals(appId, e.getAppId());
		}
		try {
			throw new TriggerNotSubscribedException(appId, testExceptionMsg);
		} catch (TriggerNotSubscribedException e) {
			assertEquals(testExceptionMsg, e.getMessage());
		}
		
		
		
	}

}

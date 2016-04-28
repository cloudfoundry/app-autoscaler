package org.cloudfoundry.autoscaler.servicebroker.exception;

import static org.junit.Assert.assertEquals;

import org.junit.Test;



public class ExceptionTest {

    @Test
    public void exceptionTest() throws InterruptedException {
    	
    	String testExceptionMsg = "This is a test exception"; 
    	
    	try {
    		throw new AlreadyBoundAnotherServiceException(testExceptionMsg);
    	} catch (AlreadyBoundAnotherServiceException e ){
    		assertEquals(testExceptionMsg, e.getMessage());
    	}

    	try {
    		throw new ProxyInitilizedFailedException(testExceptionMsg);
    	} catch (ProxyInitilizedFailedException e ){
    		assertEquals(testExceptionMsg, e.getMessage());
    	}

    	try {
    		throw new ScalingServerFailureException(testExceptionMsg);
    	} catch (ScalingServerFailureException e ){
    		assertEquals(testExceptionMsg, e.getMessage());
    	}
    	
    	try {
    		throw new ServerUrlMappingNotFoundException(testExceptionMsg);
    	} catch (ServerUrlMappingNotFoundException e ){
    		assertEquals(testExceptionMsg, e.getMessage());
    	}    	

    	try {
    		throw new ServiceBindingNotFoundException(testExceptionMsg);
    	} catch (ServiceBindingNotFoundException e ){
    		assertEquals(testExceptionMsg, e.getMessage());
    	}
    
    
    }

  
}

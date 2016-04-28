package org.cloudfoundry.autoscaler.common;

import static org.junit.Assert.assertEquals;

import org.cloudfoundry.autoscaler.common.AppInfoNotFoundException;
import org.cloudfoundry.autoscaler.common.AppNotFoundException;
import org.cloudfoundry.autoscaler.common.CloudException;
import org.cloudfoundry.autoscaler.common.InputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.InputJSONParseErrorException;
import org.cloudfoundry.autoscaler.common.InternalAuthenticationException;
import org.cloudfoundry.autoscaler.common.InternalServerErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONFormatErrorException;
import org.cloudfoundry.autoscaler.common.OutputJSONParseErrorException;
import org.cloudfoundry.autoscaler.common.PolicyNotFoundException;
import org.cloudfoundry.autoscaler.common.ServiceNotFoundException;
import org.junit.Test;



public class ExceptionTest {

    @Test
    public void exceptionTest() throws InterruptedException {
    	
    	String app_id = "bd653b9e-a7fd-4982-9366-7e233be80156";
    	String serviceName = "CF-Autoscaler";
    	String context = "Create Policy";
    	int line = 3;
    	int column = 5;
    	String AppInfoNotFoundMessage = "CWSCV6008E: The following error occurred when retrieving information for application";
    	String AppNotFoundMessage = "CWSCV6007E: The application is not found"; 
    	String CloudMessage = "CWSCV6006E: Calling CloudFoundry APIs failed";
    	String InputJSONFormatErrorMessage = "CWSCV6003E: Input JSON strings format error in the input JSON for API";
    	String InputJSONParseErrorMessage ="CWSCV6001E: The API server cannot parse the input JSON strings for API";
    	String InternalAuthenticationMessage = "CWSCV6011E: Internal Authentication failed";
    	String InternalServerErrorMessage = "CWSCV6005E: Internal server error occurred";
    	String OutputJSONFormatErrorMessage = "CWSCV6004E: Output JSON strings format error in the output JSON";
    	String OutputJSONParseErrorMessage = "CWSCV6002E: The API server cannot parse the output JSON strings";
    	String PolicyNotFoundMessage = "CWSCV6010E: Policy for App is not found";
    	String ServiceNotFoundMessage = "CWSCV6009E: Service for App is not found";
    	
    	
    	try {
    		throw new AppNotFoundException(app_id, AppNotFoundMessage);
    	} catch (AppNotFoundException e ){
    		assertEquals(AppNotFoundMessage, e.getMessage());
    		assertEquals(app_id, e.getAppId());
    	}

    	try {
    		throw new AppInfoNotFoundException(app_id, AppInfoNotFoundMessage);
    	} catch (AppInfoNotFoundException e ){
    		assertEquals(AppInfoNotFoundMessage, e.getMessage());
    		assertEquals(app_id, e.getAppId());
    	}
    
    	try {
    		throw new CloudException(CloudMessage);
    	} catch (CloudException e ){
    		assertEquals(CloudMessage, e.getMessage());
    	}
    	
    	try {
    		throw new InputJSONFormatErrorException(context, InputJSONFormatErrorMessage);
    	} catch (InputJSONFormatErrorException e ){
    		assertEquals(InputJSONFormatErrorMessage, e.getMessage());
    		assertEquals(context, e.getContext());
    	}
    	
    	try {
    		throw new InputJSONFormatErrorException(context, InputJSONFormatErrorMessage, line, column);
    	} catch (InputJSONFormatErrorException e ){
    		assertEquals(InputJSONFormatErrorMessage, e.getMessage());
    		assertEquals(context, e.getContext());
    		assertEquals(line, e.getLine());
    		assertEquals(column, e.getColumn());
    	}
    	
    	try {
    		throw new InputJSONParseErrorException(context, InputJSONParseErrorMessage);
    	} catch (InputJSONParseErrorException e ){
    		assertEquals(InputJSONParseErrorMessage, e.getMessage());
    		assertEquals(context, e.getContext());
    	}
    	
    	try {
    		throw new InternalAuthenticationException(context, InternalAuthenticationMessage);
    	} catch (InternalAuthenticationException e ){
    		assertEquals(InternalAuthenticationMessage, e.getMessage());
    		assertEquals(context, e.getContext());
    	}
    	
    	try {
    		throw new InternalServerErrorException(context, InternalServerErrorMessage);
    	} catch (InternalServerErrorException e ){
    		assertEquals(InternalServerErrorMessage, e.getMessage());
    		assertEquals(context, e.getContext());
    	}
  	
    	try {
    		throw new OutputJSONFormatErrorException(context, OutputJSONFormatErrorMessage);
    	} catch (OutputJSONFormatErrorException e ){
    		assertEquals(OutputJSONFormatErrorMessage, e.getMessage());
    		assertEquals(context, e.getContext());
    	}
  	
    	try {
    		throw new OutputJSONParseErrorException(context, OutputJSONParseErrorMessage);
    	} catch (OutputJSONParseErrorException e ){
    		assertEquals(OutputJSONParseErrorMessage, e.getMessage());
    		assertEquals(context, e.getContext());
    	}
    	
    	try {
    		throw new PolicyNotFoundException(app_id, PolicyNotFoundMessage);
    	} catch (PolicyNotFoundException e ){
    		assertEquals(PolicyNotFoundMessage, e.getMessage());
    		assertEquals(app_id, e.getAppId());
    	}
    	
    	try {
    		throw new ServiceNotFoundException(serviceName, app_id, ServiceNotFoundMessage);
    	} catch (ServiceNotFoundException e ){
    		assertEquals(ServiceNotFoundMessage, e.getMessage());
    		assertEquals(serviceName, e.getServiceName());
    		assertEquals(app_id, e.getAppId());
    	}
    	
    }

  
}
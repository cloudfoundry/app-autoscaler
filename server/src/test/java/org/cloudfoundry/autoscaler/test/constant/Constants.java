package org.cloudfoundry.autoscaler.test.constant;

public class Constants {
	
	public static final int STATUS200 = 200;
	public static final int STATUS201 = 201;
	public static final int STATUS204 = 204;
	public static final String TESTAPPID = "123456" + String.valueOf(Math.abs(Math.abs("TESTAPPID".hashCode())));
	public static final String TESTAPPNAME = "123456" + String.valueOf(Math.abs("TESTAPPNAME".hashCode()));
	public static final String TESTSERVICEID = "123456" + String.valueOf(Math.abs("TESTSERVICEID".hashCode()));
	public static final String TESTPOLICYID = "123456" + String.valueOf(Math.abs("TESTPOLICYID".hashCode()));
	public static final String TESTBINDINGID = "123456" + String.valueOf(Math.abs("TESTBINDINGID".hashCode()));
	public static final String BOUNDSTATE = "unbond";
	public static final String SERVERNAME = "AutoScaling";
	public static final String TESTORGID = "123456" + String.valueOf(Math.abs("TESTORGID".hashCode()));
	public static final String TESTSPACEID = "123456" + String.valueOf(Math.abs("TESTSPACEID".hashCode()));
	public static final String APPAUTOSCALESTATEDOCTESTID = "123456" + String.valueOf(Math.abs("APPAUTOSCALESTATEDOCTESTID".hashCode()));
	public static final String APPLICATIONDOCTESTID = "123456" + String.valueOf(Math.abs("APPLICATIONDOCTESTID".hashCode()));
	public static final String AUTOSCALERPOLICYDOCTESTID = "123456" + String.valueOf(Math.abs("AUTOSCALERPOLICYDOCTESTID".hashCode()));
	public static final String BOUNDAPPDOCTESTID = "123456" + String.valueOf(Math.abs("BOUNDAPPDOCTESTID".hashCode()));
	public static final String METRICDBSEGMENTDOCTESTID = "123456" + String.valueOf(Math.abs("METRICDBSEGMENTDOCTESTID".hashCode()));
	public static final String SCALINGHISTORYDOCTESTID = "123456" + String.valueOf(Math.abs("SCALINGHISTORYDOCTESTID".hashCode()));
	public static final String SERVICECONFIGDOCTESTID = "123456" + String.valueOf(Math.abs("SERVICECONFIGDOCTESTID".hashCode()));
	public static final String TRIGGERRECORDDOCTESTID = "123456" + TESTAPPID + "_Memory_lower_30.0";
	public static final String APPINSTANCEMATRICSDOCTESTID = "123456" + String.valueOf(Math.abs("APPINSTANCEMATRICSDOCTESTID".hashCode()));
	public static final String APPINSTANCEMETRICSDOCTESTID = "123456" + String.valueOf(Math.abs("APPINSTANCEMETRICSDOCTESTID".hashCode()));
	public static final String APPLICATIONINSTANCEDOCTESTID = "123456" + String.valueOf(Math.abs("APPLICATIONINSTANCEDOCTESTID".hashCode()));
	public static final String SERVICEINSTANCEDOCTESTID = "123456" + String.valueOf(Math.abs("SERVICEINSTANCEDOCTESTID".hashCode()));
	public static final String TESTSERVERURL = "http://localhost:9998";
	public static final String DOCUMENTNOTFOUNDERRORMSG = "nothing found on db path";
	

}

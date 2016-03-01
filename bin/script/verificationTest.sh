 #!/bin/bash
source ${basedir}/script/utils.sh

function loadRuntimeConfig(){
if [ -z $onCloud ]; then
	cfUrl=`getDefaultConfig cfUrl ${AutoScalingServerProfileDIR}/$profile.properties`
	cfDomain=${cfUrl##"api."}
	brokerUsername=`getDefaultConfig brokerUsername ${AutoScalingBrokerProfileDIR}/$profile.properties`
	brokerPassword=`getDefaultConfig brokerPassword ${AutoScalingBrokerProfileDIR}/$profile.properties`

	serverURIList=`getDefaultConfig serverURIList ${AutoScalingBrokerProfileDIR}/$profile.properties`
	apiServerURI=`getDefaultConfig apiServerURI ${AutoScalingBrokerProfileDIR}/$profile.properties`
	
	if [[ $apiServerURI == ${AutoScalingAPIName}.* ]]; then
		onCloud="y";
		hostingCustomDomain=${apiServerURI##"`eval echo $AutoScalingAPIName`."};
		hostingCFDomain=`readConfigValue hostingCFDomain "CF domain to host $componentName applications" $hostingCustomDomain`;
	else
		onCloud="n";
	fi 
fi

}

function verificationTest() {

	loadRuntimeConfig
	echo " >>> Verification test will be launched on $cfDomain"
	if  [ -z "$cfUsername" ] ; then
		echo " >>> Please input the access info for api.$cfDomain"
		cfUsername=`readConfigValue cfUsername "CF Username"`
		cfPassword=`readConfigValue cfPassword "CF Password"`
	fi
	testOrg=`readConfigValue Org "Org name to host test application " $org`
	testSpace=`readConfigValue Space "Space name to host test application" $space`

	${basedir}/script/test/launchTest.sh $serviceName $cfDomain $cfUsername $cfPassword $testOrg $testSpace

}
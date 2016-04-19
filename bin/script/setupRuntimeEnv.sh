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

function launchRuntime() {


if [ $onCloud == "n" ]; then
	echo " >>> Please setup your runtime environment runtime MANUALLY, and align with previous setting "
	echo " serverURIList : $serverURIList"
	echo " apiServerURI : $apiServerURI"
    read -p "Press Any key to continue when runtime environment is launched ... " input
   
else
	echo " >>> Now, the script will push $componentName to $hostingCFDomain"
	echo " >>> Please input the access info for api.$hostingCFDomain "
	cfUsername=`readConfigValue cfUsername "CF Username"`
	cfPassword=`readConfigValue cfPassword "CF Password"`
	org=`readConfigValue Org "CF Org to host $componentName"`
	space=`readConfigValue Space "CF Space to host $componentName"`

	cf login -a https://api.$hostingCFDomain -u $cfUsername -p $cfPassword -o $org -s $space  --skip-ssl-validation

	pushMavenPackageToCF ${AutoScalingServerName} ${AutoScalingServerProjectDirName}
	pushMavenPackageToCF ${AutoScalingAPIName} ${AutoScalingAPIProjectDirName}
	pushMavenPackageToCF ${AutoScalingBrokerName} ${AutoScalingBrokerProjectDirName}
	
fi

}

function registerService(){

if [ $onCloud == "n" ]; then
	read -p "Please input URL of service broker " brokerURI
else
	brokerURI="http://"${AutoScalingBrokerName}.$hostingCustomDomain
fi


if [ "$cfDomain" != "$hostingCFDomain" ] || [ -z "$cfUsername" ] ; then
	echo " >>> Please input the access info of api.$cfDomain"
	cfUsername=`readConfigValue cfUsername "CF Username"`
	cfPassword=`readConfigValue cfPassword "CF Password"`
	org=`readConfigValue Org "CF Org which is accessible for $cfUsername"`
	space=`readConfigValue Space "CF Space which is accessible for $cfUsername"`
fi

cf login -a https://api.$cfDomain -u $cfUsername -p $cfPassword -o $org -s $space --skip-ssl-validation

cf marketplace -s $serviceName

if [ $? -ne 0 ]; then
	echo "cf create-service-broker $serviceName<brokerUserName> <brokerPassword> $brokerURI"
	cf create-service-broker $serviceName $brokerUsername $brokerPassword $brokerURI
	cf enable-service-access $serviceName
else 
	echo "cf update-service-broker $serviceName <brokerUserName> <brokerPassword> $brokerURI"
	cf update-service-broker $serviceName $brokerUsername $brokerPassword $brokerURI
fi
}


function setupRuntimeEnv(){
	loadRuntimeConfig
	launchRuntime
	registerService

}

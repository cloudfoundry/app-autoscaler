 #!/bin/bash
source ${basedir}/script/utils.sh

function loadRuntimeConfig(){
if [ -z $onCloud ]; then
	cfUrl=`getDefaultConfig cfUrl ${AutoScalingProfileDIR}/$profile.properties`
	cfDomain=${cfUrl##"api."}
	brokerUsername=`getDefaultConfig brokerUsername ${AutoScalingProfileDIR}/$profile.properties`
	brokerPassword=`getDefaultConfig brokerPassword ${AutoScalingProfileDIR}/$profile.properties`

	serverURIList=`getDefaultConfig serverURIList ${AutoScalingProfileDIR}/$profile.properties`
	apiServerURI=`getDefaultConfig apiServerURI ${AutoScalingProfileDIR}/$profile.properties`

	if [[ $apiServerURI == ${AutoScalingAPIName}.* ]]; then
		onCloud="y";
		hostingCustomDomain=${apiServerURI##$AutoScalingAPIName}
		hostingCustomDomain=${hostingCustomDomain:1}
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
	echo " >>> Pushing $componentName to $cfUrl"
	cfUsername=`readConfigValue cfUsername "CF Username"`
	cfPassword=`readConfigValue cfPassword "CF Password"`
	org=`readConfigValue Org "Organization for $componentName"`
	space=`readConfigValue Space "Space for $componentName"`

	cf login -a https://$cfUrl -u $cfUsername -p $cfPassword -o $org -s $space  --skip-ssl-validation
	if [[ $? != 0 ]]; then
		exit $?
	fi

	pushMavenPackageToCF ${AutoScalingServerName} ${AutoScalingServerProjectDirName}
	pushMavenPackageToCF ${AutoScalingAPIName} ${AutoScalingAPIProjectDirName}
	pushMavenPackageToCF ${AutoScalingBrokerName} ${AutoScalingBrokerProjectDirName}
fi
}

function registerService(){

if [ $onCloud == "n" ]; then
	read -p "Please input URL of service broker " brokerURI
else
	brokerURI="https://"${AutoScalingBrokerName}.$hostingCustomDomain
fi

hostingCFDomain
if [ "$cfDomain" != "$hostingCustomDomain" ] || [ -z "$cfUsername" ] ; then
	echo " >>> Please input the access info of $cfUrl"
	cfUsername=`readConfigValue cfUsername "CF Username"`
	cfPassword=`readConfigValue cfPassword "CF Password"`
	org=`readConfigValue Org "Organization for $componentName"`
	space=`readConfigValue Space "Space for $componentName"`
fi

cf login -a https://$cfUrl -u $cfUsername -p $cfPassword -o $org -s $space  --skip-ssl-validation

cf marketplace -s $serviceName

if [ $? -ne 0 ]; then
	echo "cf create-service-broker $serviceName <brokerUserName> <brokerPassword> $brokerURI"
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

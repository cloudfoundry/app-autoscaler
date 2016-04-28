 #!/bin/bash
source ${basedir}/script/utils.sh


function createProfile(){
read -p " >>> Define customized environment (default: sample): " profile
if [ -z "$profile" ]; then
	profile="sample"
fi

profileExist=true;
AutoScalingProfileDIR="${basedir}/../profiles/"
if [ ! -f ${AutoScalingProfileDIR}/$profile.properties ]; then
	cp ${AutoScalingProfileDIR}/${SampleProfile}.properties ${AutoScalingProfileDIR}/$profile.properties
	profileExist=false;
fi
}

function readConfiguration() {

echo " >>> Loading default values from $profile "
echo " >>> Now, you can edit the entries according to your runtime"

cfUrl=`readConfigValue cfUrl "Cloudfoundry API URL in which you want to register $serviceName service "`
cfDomain=${cfUrl##"api."}
cfClientID=`readConfigValue cfClientId "CF Client ID for $cfDomain"`
cfClientSecret=`readConfigValue cfClientSecret "CF Client Secret for $cfDomain"`

couchdbHost=`readConfigValue couchdbHost "Hostname of Couchdb"`
couchdbPort=`readConfigValue couchdbPort "Port of Couchdb"`
couchdbUsername=`readConfigValue couchdbUsername "Username of Couchdb"`
couchdbPassword=`readConfigValue couchdbPassword "Password of Couchdb"`

brokerUsername=`readConfigValue brokerUsername "Broker username to register service"`
brokerPassword=`readConfigValue brokerPassword "Broker password to register service"`

internalAuthUsername=`readConfigValue internalAuthUsername "The username for http-basic authorization between different project of $componentName"`
internalAuthPassword=`readConfigValue internalAuthPassword "The password for http-basic authorization between different project of $componentName"`

while true; do
    read -p " Would you like to host $componentName on Cloudfoundry? (default: y): "  onCloud
    case $onCloud in
        [Yy]* ) onCloud=y;
 				hostingCustomDomain=`readConfigValue hostingCustomDomain "Domain to host $componentName" $cfDomain`;
 				serverURIList="$AutoScalingServerName.$hostingCustomDomain";
 				apiServerURI="$AutoScalingAPIName.$hostingCustomDomain";
 				break;;
        [Nn]* ) onCloud=n;
 				serverURIList=`readConfigValue serverURIList "$componentName Server url list"`;
				apiServerURI=`readConfigValue apiServerURI "$componentName API url"`;
				break;;
        * ) echo "Please answer yes or no.";;
    esac
done

}

function setConfiguration() {
echo > ${AutoScalingProfileDIR}/profiles/$profile.properties << EOF
#Cloud Foundry settings
cfUrl=${cfUrl}
cfClientId=${cfClientID}
cfClientSecret=${cfClientSecret}

# http basic auth between AutoScaler components
internalAuthUsername=${internalAuthUsername}
internalAuthPassword=${internalAuthPassword}

#service.name
service.name=cf-autoscaler

#broker credentials
brokerUsername=${brokerUsername}
brokerPassword=${brokerPassword}

# URLs
serverURIList=${serverURIList}
apiServerURI=${apiServerURI}

#metrics settings
reportInterval=120

#couchdb settings

couchdbHost=${couchdbHost}
couchdbPort=${couchdbPort}
couchdbUsername=${couchdbUsername}
couchdbPassword=${couchdbPassword}
couchdbServerDBName=couchdb-scaling
couchdbMetricDBPrefix=couchdb-scalingmetric
couchdbBrokerDBName=couchdb-scalingbroker

EOF
}

function showConfiguration() {

promptHint " >>> display $componentName configuration: "
showConfigValue cfUrl ${AutoScalingProfileDIR}/$profile.properties
showConfigValue cfClientId ${AutoScalingProfileDIR}/$profile.properties
showConfigValue cfClientSecret ${AutoScalingProfileDIR}/$profile.properties
showConfigValue couchdbHost ${AutoScalingProfileDIR}/$profile.properties
showConfigValue couchdbPort ${AutoScalingProfileDIR}/$profile.properties
showConfigValue couchdbUsername ${AutoScalingProfileDIR}/$profile.properties
showConfigValue couchdbPassword ${AutoScalingProfileDIR}/$profile.properties
showConfigValue internalAuthUsername ${AutoScalingProfileDIR}/$profile.properties
showConfigValue internalAuthPassword ${AutoScalingProfileDIR}/$profile.properties

showConfigValue brokerUsername ${AutoScalingProfileDIR}/$profile.properties
showConfigValue brokerPassword ${AutoScalingProfileDIR}/$profile.properties

showConfigValue serverURIList ${AutoScalingProfileDIR}/$profile.properties
showConfigValue apiServerURI ${AutoScalingProfileDIR}/$profile.properties

echo
}


function configProfile(){
if [ "$profileExist" == "true" ]; then
	reuseExistingProfile=$(confirmYes " Would you like to reuse your configuration in $profile? (y/n) ")
	if [ $reuseExistingProfile == "n" ]; then
		readConfiguration
		setConfiguration
	fi
else
	readConfiguration
	setConfiguration
fi

showConfiguration
confirmConfiguration=$(confirmYes " Proceed with configuration? (y/n) ")
if [ $confirmConfiguration == "n" ]; then
	exit 0
fi

}



function configProject() {
	createProfile
	configProfile
}

 #!/bin/bash
source ${basedir}/script/utils.sh


function createProfile(){
read -p " >>> Define customized Maven profile name: (default: test): " profile
if [ -z "$profile" ]; then
	profile="test"
fi

profileExist=true;
AutoScalingServerProfileDIR="${basedir}/../$AutoScalingServerProjectDirName/profiles/"
if [ ! -f ${AutoScalingServerProfileDIR}/$profile.properties ]; then
	cp ${AutoScalingServerProfileDIR}/${SampleProfile}.properties ${AutoScalingServerProfileDIR}/$profile.properties
	profileExist=false;
fi

AutoScalingAPIProfileDIR="${basedir}/../$AutoScalingAPIProjectDirName/profiles/"
if [ ! -f ${AutoScalingAPIProfileDIR}/$profile.properties ]; then
	cp ${AutoScalingAPIProfileDIR}/${SampleProfile}.properties ${AutoScalingAPIProfileDIR}/$profile.properties
	profileExist=false;
fi

AutoScalingBrokerProfileDIR="${basedir}/../$AutoScalingBrokerProjectDirName/profiles/"
if [ ! -f ${AutoScalingBrokerProfileDIR}/$profile.properties ]; then
	cp ${AutoScalingBrokerProfileDIR}/${SampleProfile}.properties ${AutoScalingBrokerProfileDIR}/$profile.properties
	profileExist=false;
fi

}


function addProfileDefinition(){
echo " >>> Please paste the following into ~/.m2/settings.xml <profiles> section"  
echo "	<profile>
            <id>$profile</id>
            <properties>
                <build.profile.id>$profile</build.profile.id>
            </properties>
        </profile>
    "

echo " >>> Press any key to continue ..." 
read input


}

function readConfiguration() {

echo " >>> Load default value from Maven profile $profile "
echo " >>> Now, you can edit the entries according to your runtime envrionment" 

cfUrl=`readConfigValue cfUrl "Cloudfoundry API URL in which you want to register $serviceName service "`
cfDomain=${cfUrl##"api."}
cfClientID=`readConfigValue cfClientId "CF Client ID for $cfDomain"`
cfClientSecretBase64Encoded=`readConfigValueBase64Encoded cfClientSecretBase64Encoded "CF Client Secret for $cfDomain"`

couchdbHost=`readConfigValue couchdbHost "Hostname of Couchdb"`
couchdbPort=`readConfigValue couchdbPort "Port of Couchdb"`
couchdbUsername=`readConfigValue couchdbUsername "Username of Couchdb"`
couchdbPasswordBase64Encoded=`readConfigValueBase64Encoded couchdbPasswordBase64Encoded "Password of Couchdb"`

brokerUsername=`readConfigValue brokerUsername "Broker username to register service"`
brokerPassword=`readConfigValue brokerPassword "Broker password to register service"`

internalAuthUsername=`readConfigValue internalAuthUsername "The username for http-basic authorization between different project of $componentName"`
internalAuthPassword=`readConfigValue internalAuthPassword "The password for http-basic authorization between different project of $componentName"`
internalAuthTokenBase64Encoded=`echo -n "$internalAuthUsername:$internalAuthPassword" | base64 `

while true; do
    read -p " Would you like to host $componentName applications on Cloudfoundry? (default: y): "  onCloud
    case $onCloud in
        [Yy]* ) onCloud=y;
 				hostingCFDomain=`readConfigValue hostingCFDomain "CF domain to host $componentName applications" $cfDomain`;
 				hostingCustomDomain=`readConfigValue hostingCustomDomain "CF custom domain of the hosting applications " $cfDomain`;
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
setProperties ${AutoScalingServerProjectDirName} cfUrl ${cfUrl}
setProperties ${AutoScalingServerProjectDirName} cfClientId ${cfClientID}
setProperties ${AutoScalingServerProjectDirName} cfClientSecretBase64Encoded ${cfClientSecretBase64Encoded}
setProperties ${AutoScalingServerProjectDirName} couchdbHost ${couchdbHost}
setProperties ${AutoScalingServerProjectDirName} couchdbPort ${couchdbPort}
setProperties ${AutoScalingServerProjectDirName} couchdbUsername ${couchdbUsername}
setProperties ${AutoScalingServerProjectDirName} couchdbPasswordBase64Encoded ${couchdbPasswordBase64Encoded}
setProperties ${AutoScalingServerProjectDirName} internalAuthTokenBase64Encoded ${internalAuthTokenBase64Encoded}


setProperties ${AutoScalingAPIProjectDirName} cfUrl ${cfUrl}
setProperties ${AutoScalingAPIProjectDirName} cfClientId ${cfClientID}
setProperties ${AutoScalingAPIProjectDirName} cfClientSecretBase64Encoded ${cfClientSecretBase64Encoded}
setProperties ${AutoScalingAPIProjectDirName} internalAuthUsername ${internalAuthUsername}
setProperties ${AutoScalingAPIProjectDirName} internalAuthPassword ${internalAuthPassword}

setProperties ${AutoScalingBrokerProjectDirName} couchdbHost ${couchdbHost}
setProperties ${AutoScalingBrokerProjectDirName} couchdbPort ${couchdbPort}
setProperties ${AutoScalingBrokerProjectDirName} couchdbUsername ${couchdbUsername}
setProperties ${AutoScalingBrokerProjectDirName} couchdbPasswordBase64Encoded ${couchdbPasswordBase64Encoded}
setProperties ${AutoScalingBrokerProjectDirName} brokerUsername ${brokerUsername}
setProperties ${AutoScalingBrokerProjectDirName} brokerPassword ${brokerPassword}

setProperties ${AutoScalingBrokerProjectDirName} serverURIList ${serverURIList}
setProperties ${AutoScalingBrokerProjectDirName} apiServerURI ${apiServerURI}
setProperties ${AutoScalingBrokerProjectDirName} internalAuthUsername ${internalAuthUsername}
setProperties ${AutoScalingBrokerProjectDirName} internalAuthPassword ${internalAuthPassword}

}

function showConfiguration() {

promptHint " >>> display $componentName $AutoScalingServerProjectDirName configuration: "
showConfigValue cfUrl ${AutoScalingServerProfileDIR}/$profile.properties
showConfigValue cfClientId ${AutoScalingServerProfileDIR}/$profile.properties
showConfigValueBase64Encoded cfClientSecretBase64Encoded ${AutoScalingServerProfileDIR}/$profile.properties cfClientSecret
showConfigValue couchdbHost ${AutoScalingServerProfileDIR}/$profile.properties
showConfigValue couchdbPort ${AutoScalingServerProfileDIR}/$profile.properties
showConfigValue couchdbUsername ${AutoScalingServerProfileDIR}/$profile.properties
showConfigValueBase64Encoded couchdbPasswordBase64Encoded ${AutoScalingServerProfileDIR}/$profile.properties couchdbPassword
showConfigValueBase64Encoded internalAuthTokenBase64Encoded ${AutoScalingServerProfileDIR}/$profile.properties internalAuthToken


promptHint " >>> display $componentName $AutoScalingAPIProjectDirName configuration: "
showConfigValue cfUrl ${AutoScalingAPIProfileDIR}/$profile.properties
showConfigValue cfClientId ${AutoScalingAPIProfileDIR}/$profile.properties
showConfigValueBase64Encoded cfClientSecretBase64Encoded ${AutoScalingAPIProfileDIR}/$profile.properties cfClientSecret
showConfigValue internalAuthUsername ${AutoScalingAPIProfileDIR}/$profile.properties
showConfigValue internalAuthPassword ${AutoScalingAPIProfileDIR}/$profile.properties

promptHint " >>> display $componentName $AutoScalingBrokerProjectDirName configuration: "
showConfigValue couchdbHost ${AutoScalingBrokerProfileDIR}/$profile.properties
showConfigValue couchdbPort ${AutoScalingBrokerProfileDIR}/$profile.properties
showConfigValue couchdbUsername ${AutoScalingBrokerProfileDIR}/$profile.properties
showConfigValueBase64Encoded couchdbPasswordBase64Encoded ${AutoScalingBrokerProfileDIR}/$profile.properties couchdbPassword

showConfigValue brokerUsername ${AutoScalingBrokerProfileDIR}/$profile.properties
showConfigValue brokerPassword ${AutoScalingBrokerProfileDIR}/$profile.properties

showConfigValue internalAuthUsername ${AutoScalingBrokerProfileDIR}/$profile.properties
showConfigValue internalAuthPassword ${AutoScalingBrokerProfileDIR}/$profile.properties

showConfigValue serverURIList ${AutoScalingBrokerProfileDIR}/$profile.properties
showConfigValue apiServerURI ${AutoScalingBrokerProfileDIR}/$profile.properties


echo
}


function configProfile(){

if [ "$profileExist" == "true" ]; then
	reuseExistingProfile=$(confirmYes " Would you like to reuse all configuration in profile $profile? (y/n) ")
	if [ $reuseExistingProfile == "n" ]; then
		readConfiguration
		setConfiguration
	fi
else
	addProfileDefinition
	readConfiguration
	setConfiguration
fi


showConfiguration
confirmConfiguration=$(confirmYes " Proceed $componentName with above configuration? (y/n) ")
if [ $confirmConfiguration == "n" ]; then
	exit 0
fi

}



function configProject() {
	createProfile
	configProfile
}
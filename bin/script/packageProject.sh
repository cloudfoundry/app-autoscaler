 #!/bin/bash
source ${basedir}/script/utils.sh

function configForEclipse(){

covertRequired=$(confirmNo "Would you like to convert $componentName Maven project for Eclipse? (default: n) ")

if [ $covertRequired == "y" ]; then
	echo " >>> Execute \"mvn eclipse:eclipse -Dwtpversion=2.0\" for $componentName" 
 	echo " >>> It will take time for the first run. Please wait ..."
 	configMavenForEclipse ${AutoScalingServerProjectDirName}
 	configMavenForEclipse ${AutoScalingAPIProjectDirName}
 	configMavenForEclipse ${AutoScalingBrokerProjectDirName}
 fi

}


function packageProject() {

configForEclipse
echo " >>> Execute \"mvn clean package\" for $componentName" 
echo " >>> It will take time for the first run. Please wait ..."
packageMavenProject ${AutoScalingServerProjectDirName} 
packageMavenProject ${AutoScalingAPIProjectDirName} 
packageMavenProject ${AutoScalingBrokerProjectDirName} 

}
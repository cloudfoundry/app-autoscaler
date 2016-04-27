 #!/bin/bash
source ${basedir}/default.properties

if [ "$(uname)" == "Darwin" ]; then
	SHELL="mac"
else
	SHELL="unix"
fi


function confirmYes(){
	echo -n $1 > /dev/stderr
	read  yn
	case $yn in
        [Nn]* ) yn=n;;
        * ) yn=y;;
	esac
	echo $yn
}

function confirmNo(){
	echo -n $1 > /dev/stderr
	read yn
	case $yn in
        [Yy]* ) yn=y;;
        * ) yn=n;;
	esac

	echo $yn
}

function jumpToNextStep(){
	read -p "Would you like to SKIP this step? (n/y):" yn
	case $yn in
        [Yy]* ) yn=y;;
        * ) yn=n;;
	esac

	echo $yn
}

function continueNextStep {
    read -p " Press Any key to continue .." 
}

function promptHint () {
	defaultColor="\033[0m"
	highlightColor="\033[32m"
	echo -e ${highlightColor} "\n" $1 ${defaultColor} ":" > /dev/stderr
}


function promptInput () {
	defaultColor="\033[0m"
	highlightColor="\033[32m"
	if [ -z "$2" ]; then
		echo -e -n ${defaultColor} "Please define" ${highlightColor} $1 ${defaultColor} ":" > /dev/stderr
	else
	 	echo -e -n ${defaultColor} "Please define" ${highlightColor} $1 ${defaultColor} "[default:$2] :" > /dev/stderr
	fi
}

function getDefaultConfig() {
	local key=$1
	local filename=$2
	local value;
	if [ -z $filename ]; then
		value=`cat ${basedir}/default.properties \
					$AutoScalingServerProfileDIR/$profile.properties \
					$AutoScalingAPIProfileDIR/$profile.properties \
					$AutoScalingBrokerProfileDIR/$profile.properties \
					| grep "$key"  | head -n 1 |  awk -F "$key=" '{print $2}'`
	else
		value=`cat $filename | grep "$key"  | head -n 1 | awk -F "$key=" '{print $2}'`
	fi

	echo $value
}


function showConfigValue() {
	echo "$1="`getDefaultConfig $1 $2`
}



function readConfigValue() {
	local key=$1
	local description=$2
	local defaultValue=$3
	local prompt;
	if [ -z $defaultValue ]; then
		defaultValue=`getDefaultConfig $key`
	fi 
	prompt=`promptInput "$description" $defaultValue`
	
	read -p "$prompt" inputValue
	if [ -z $inputValue ]; then
		inputValue=$defaultValue
	fi
	echo $inputValue

}


function setProperties() {
	local projectDirName=${basedir}/../$1
	local key=$2
	local value=$3
	local propertieFile=$projectDirName/profiles/$profile.properties

	#echo " >>> set \"$key=$value\" in $propertieFile" > /dev/stderr
	if [ -z `cat $propertieFile | grep $key` ]; then
		echo "$key=$value" >> $propertieFile
	else
		if [ "$SHELL" == "mac" ]; then
			sed -i '' "/$key/d" $propertieFile
			echo "$key=$value" >> $propertieFile
		else
			sed -i "/$key/d" $propertieFile
			echo "$key=$value" >> $propertieFile
		fi
	fi 
}

function configMavenForEclipse(){

	local projectDirName=${basedir}/../$1
	cd $projectDirName
	mvn eclipse:eclipse -Dwtpversion=2.0 > build.log
	if [ $? -eq 0 ]; then
		echo " >>> Convert project $projectDirName Successfully" 
	else
		cat build.log > /dev/stderr
		echo " >>> Convert project $projectDirName Failed" 
		rm build.log
		exit 1
	fi
	rm build.log
}


function packageMavenProject() {
	local projectDirName=${basedir}/../$1
	local warFileName=$1

	cd $projectDirName
	mvn test -Punittest
	if [[ $? -eq 0 ]]; then
		echo ">>>>>>>>>>>>> Unit test Successfully"
	else 
		echo ">>>>>>>>>>>>> Unit test Failed"
		exit 1
	fi
	mvn clean package -P$profile -Dmaven.test.skip=true > build.log
	if [ $? -eq 0 ]; then
		echo " >>> Package $projectDirName/build/$warFileName.war Successfully" 
	else
		cat build.log > /dev/stderr
		echo " >>> Package $projectDirName/build/$warFileName.war Failed" 
		rm build.log
		exit 1
	fi
	rm build.log

}

function pushMavenPackageToCF() {
	local appName=$1
	local warDirName=${basedir}/../$2/target
	local warFileName=$2-1.0-SNAPSHOT.war
	echo " >>> Push file $warDirName/$warFileName "
	cf push $appName -p $warDirName/$warFileName -d $hostingCustomDomain
		
}

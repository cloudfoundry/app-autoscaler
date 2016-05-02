 #!/bin/bash

basedir=$(cd "$(dirname "$0")"; pwd)
source ${basedir}/script/utils.sh
source ${basedir}/default.properties

if [ "$(uname)" == "Darwin" ]; then
	SHELL="mac"
else
	SHELL="unix"
fi

promptHint "Prerequisite software:"
echo " >>> Maven 3.0 or above"
echo " >>> CloudFoundry command line 'cf' v6 or above"
echo " >>> \"curl\" supported in SHELL "

promptHint "Step 1: Configure $componentName"
source $basedir/script/configProject.sh
configProject

promptHint "Step 2: Package $componentName"
source $basedir/script/packageProject.sh
packageProject

promptHint "Step 3: Setup $componentName Runtime Environment"
source $basedir/script/setupRuntimeEnv.sh
setupRuntimeEnv

# promptHint "Step 4: Verify $componentName"
# source $basedir/script/verificationTest.sh
# verificationTest

exit 0

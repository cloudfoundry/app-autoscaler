#!/bin/bash

function do_create_service() {

    echo "============== Create $serviceName service instance $SEVICE_INSTANCE_NAME ============================================"
    local SEVICE_INSTANCE_NAME=$1

    cf create-service $serviceName free $SEVICE_INSTANCE_NAME
    if [ $? -ne 0 ]; then
        echo "create service failed"
        return 254
    fi
    echo "============== Create $serviceName service instance $SEVICE_INSTANCE_NAME successfully ============================================"
    
}

function do_bind_service() {
    local TestAPP_NAME=$1
    local SEVICE_INSTANCE_NAME=$2
    echo "============== Bind $serviceName service instance $SEVICE_INSTANCE_NAME to ${TestAPP_NAME} ============================================"
    cf bind-service $TestAPP_NAME $SEVICE_INSTANCE_NAME
    if [ $? -ne 0 ]; then
        echo "bind service failed"
        return 254
    fi
    echo "============== Bind $serviceName service instance $SEVICE_INSTANCE_NAME to ${TestAPP_NAME} successfully ============================================"
 
}


function do_unbind_service() {

    local TestAPP_NAME=$1
    local SEVICE_INSTANCE_NAME=$2

   echo "============== Unbind $serviceName service instance $SEVICE_INSTANCE_NAME from ${TestAPP_NAME} ============================================"
    cf unbind-service $TestAPP_NAME $SEVICE_INSTANCE_NAME
    if [ $? -ne 0 ]; then
        echo "unbind service failed"
        return 254
    fi
    echo "============== Unbind $serviceName service instance $SEVICE_INSTANCE_NAME from ${TestAPP_NAME} successfully ============================================"
    
}

function do_delete_service() {

    local SEVICE_INSTANCE_NAME=$1
    echo "============== Delete $serviceName service instance $SEVICE_INSTANCE_NAME ============================================"
    cf delete-service $SEVICE_INSTANCE_NAME -f
    if [ $? -ne 0 ]; then
        echo "delete service failed"
        return 254
    fi
    echo "============== Delete $serviceName service instance $SEVICE_INSTANCE_NAME successfully ============================================"
    
}

function test_servicebroker_API() {
    local returncode=0
    do_create_service $SEVICE_INSTANCE_NAME
    ((returncode+=$?))
    do_bind_service $TestAPP_NAME  $SEVICE_INSTANCE_NAME
    ((returncode+=$?))
    do_unbind_service $TestAPP_NAME  $SEVICE_INSTANCE_NAME
    ((returncode+=$?))
    do_delete_service $SEVICE_INSTANCE_NAME
    ((returncode+=$?))
    return $returncode
 }

function do_getAPIURL() {
    local TestAPP_NAME=$1
    echo `cf env $TestAPP_NAME | grep "api_url" | awk '{print $2}' | cut -d "," -f1 | cut -d "\"" -f2`
}

function setup_PublicAPI_TestEnv() {
    do_create_service $SEVICE_INSTANCE_NAME
    do_bind_service $TestAPP_NAME  $SEVICE_INSTANCE_NAME
    cf restart $TestAPP_NAME
}


function setup_PublicAPI_TestConfig() {

    oauth_token=$(cf oauth-token | grep bearer)
    echo "Oauth-token = ${oauth_token}"

    app_guid=$(cf app $TestAPP_NAME --guid)
    echo "app_guid = ${app_guid}"

    API_url=$(do_getAPIURL $TestAPP_NAME)
    echo "API_url = $API_url "

    responseFile=$basedir"/response.txt"
    echo "curl response is logged in file : $responseFile"
 
}

function cleanup_PublicAPI_TestConfig(){
    rm -rf $responseFile
}


function do_delete_policy() {

    echo "====== Begin delete policy test ======================================================"

    delete_policy_curl_cmd="curl -k $policy_url -X 'DELETE'  -H 'Accept:application/json' -H 'Authorization:$oauth_token' -s -o $responseFile -w '%{http_code}\n'"
    echo "try to clean existing policy for $TestAPP_NAME"
    echo "$delete_policy_curl_cmd"
    eval status_code=\$\($delete_policy_curl_cmd\)
    if [[ $status_code -eq 200 ]]; then
        echo "delete policy succeed"
    else
        if [[ $status_code -eq 404 ]]; then
            if cat $responseFile | grep "CWSCV6010E"  &> /dev/null ; then
                echo "== policy for this app does not exist !!!"
            else
                echo "========== ERROR: other error happend during clean current policy in create policy test"
                return 254
            fi        
        else
            echo "delete policy faild with status_code:${status_code}"
            return 254
        fi
    fi

}


function do_create_policy() {

    policyFile=$1

    echo "====== Begin create policy test for file $policyFile ======================================================"

    create_policy_curl_cmd="curl -k $policy_url -X 'PUT' -H 'Content-Type:application/json'  -H 'Accept:application/json' -H 'Authorization:$oauth_token' --data-binary @$policyFile -s -o $responseFile -w '%{http_code}\n'"
    echo "== create_policy_curl_cmd  $create_policy_curl_cmd"    #
    eval status_code=\$\($create_policy_curl_cmd\)
    echo "status_code = $status_code"
    if [[ $status_code -eq 201 ]]; then
            echo "status_code is 201 as expected"
            if cat $responseFile | grep "policyId"  &> /dev/null ; then
                echo "== create policy for ${policyFile} success !!!"
            else
                echo "========== ERROR: other error happend during create policy test for ${name} in create policy test"
                return 254
            fi        
    else
            echo "========== ERROR: create policy fail with create policy test for ${name} and status_code: ${status_code}"
            exit 253
    fi  
}

function do_update_policy() {

    policyFile=$1
    echo "====== Begin update policy test for file $policyFile ======================================================"

    update_policy_curl_cmd="curl -k $policy_url -X 'PUT' -H 'Content-Type:application/json'  -H 'Accept:application/json' -H 'Authorization:$oauth_token' --data-binary @$policyFile  -s -o $responseFile -w '%{http_code}\n'"
    echo "== update_policy_curl_cmd  $update_policy_curl_cmd"    #
    eval status_code=\$\($update_policy_curl_cmd\)
    echo "status_code = $status_code"

    if [[ $status_code -eq 200 ]]; then
        echo "update policy succeed"
    else
        echo "update policy faild with status_code:${status_code}"
        return 254
    fi
}


function do_get_policy() {
    echo "====== get policy test  ======================================================"

    get_curl_cmd="curl -k $policy_url -X 'GET'  -H 'Accept:application/json' -H 'Authorization:$oauth_token' -s -o $responseFile -w '%{http_code}\n'"
    echo "try to get current policy for get policy test"
    eval status_code=\$\($get_curl_cmd\)
    if [[ $status_code -eq 200 ]]; then
        echo "== get policy succeed as expected!!!"
    else
        if [[ $status_code -ne 404 ]]; then
            echo "========== ERROR: other error happend during get policy with status_code:${status_code}"
            return 254
        else
            echo "========== ERROR: Fail to get the policy with status_code:${status_code}"
            return 254
        fi
    fi
}

function do_get_metric() {

    metrics_curl_cmd="curl -k \"$metrics_cf_url\" -X 'GET' -H 'Accept:application/json' -H 'Authorization:$oauth_token' -s -o $responseFile -w '%{http_code}\n'"
    echo $metrics_curl_cmd
    
    eval status_code=\$\($metrics_curl_cmd\)
    if [[ $status_code -ne 200 ]]; then
            echo "========== ERROR: error happend during get metrics in get metrics test with status_code:${status_code}"
            return 254
    fi
    if cat $responseFile | grep "data"  &> /dev/null ; then
            echo "== get metrics succeed as expected!!!"
    else
            echo "========== ERROR: other error happend during get metrics in get metrics test"
            return 254
    fi

    echo "====== finished get metrics test ======================================================" 

}


function do_get_scaling_history(){
    

    history_curl_cmd="curl -k \"$history_cf_url\" -X 'GET' -H 'Accept:application/json' -H 'Authorization:$oauth_token' -s -o $responseFile -w '%{http_code}\n'"
    echo $history_curl_cmd
    eval status_code=\$\($history_curl_cmd\)
    if [[ $status_code -ne 200 ]]; then
        echo "========== ERROR: error happend during get scaling history in get scaling history test with status_code:${status_code}"
        return 254
    fi
    if cat $responseFile | grep "data"  &> /dev/null ; then
        echo "== get scaling history succeed as expected!!!"
    else
        echo "========== ERROR: other error happend during get scaling history in get scaling history test"
        return 254
    fi

  
}

function test_Policy_API() {
    setup_PublicAPI_TestConfig

    local returncode=0
    ((returncode+=$?))
    policy_url="$API_url/v1/autoscaler/apps/$app_guid/policy" 
    echo "====== Test for Create/Get/Update/Delete Policy API =============="
    do_delete_policy 
    ((returncode+=$?))
    do_create_policy ${Resource_DIR}/file/policy/all.json
    ((returncode+=$?))
    do_get_policy
    ((returncode+=$?))
    do_update_policy ${Resource_DIR}/file/policy/dynamic.json
    ((returncode+=$?))
    do_update_policy ${Resource_DIR}/file/policy/recurringSchedule.json
   ((returncode+=$?))
    do_update_policy ${Resource_DIR}/file/policy/specificDate.json
   ((returncode+=$?))
    do_delete_policy 

    cleanup_PublicAPI_TestConfig
    return $returncode

}


function test_Metric_API() {
    setup_PublicAPI_TestConfig

    local returncode=0;
    metrics_cf_url="$API_url/v1/autoscaler/apps/$app_guid/metrics"
    echo "====== Test for Get Metric API =============="
    do_get_metric
    returncode=$?
 
    cleanup_PublicAPI_TestConfig
    return $returncode
}

function test_History_API() {

    setup_PublicAPI_TestConfig
    
    local returncode=0
    history_cf_url="$API_url/v1/autoscaler/apps/$app_guid/scalinghistory"
    echo "====== Test for Get History API =============="
    do_get_scaling_history
    returncode=$?
 
    cleanup_PublicAPI_TestConfig
    return $returncode
}


function checkInstance(){
    echo  -e "\nChecking app instances :"                                                                                                        
    for (( i=0; i<10; i++ )) ;                                                                                                                      
        do                                                                                                                                              
            instances=`cf app $TestAPP_NAME  |grep 'instances:' | awk '{ print $2 }'`                                                                      
            echo "  >>> `date` Running instance number: $instances"                                                                                                 
                runninginstances=`echo $instances | awk -F '/' '{print $1}'`            
            if [ -z "$runninginstances" ]; then
                echo " >>> `date` Fail to get running instances number"                                                                                                                                                                                                                                                           
            elif [ $runninginstances -gt 1 ];then                                                                                                                   
                break                                                                                                                                       
            fi                                                                                                                                            
            sleep 120                                                                                                                                     
        done                                                                                                                                            
                                                                                                                                                   
    if [ $i -ge 10 ]; then                                                                                                                          
        return -1                                                                                                                                    
    else
        return 0                                                                                                                  
    fi              
}




function doCheckScaleByMetrics(){

    echo " >>> Test Dynamic Scaling-out with Metric: Memory"
    local returncode=0

    echo " >>> Please create an application whose memory allocation will be varied by workload. "
    echo " >>> Please input the file path of the application package: " 
    read customizedAppFilePath

    echo " >>> Push the application $customizedAppFilePath "
    cf push  $TestAPP_NAME -p $customizedAppFilePath  --random-route 

    cf scale $TestAPP_NAME -i 1 > /dev/stderr
    setup_PublicAPI_TestConfig > /dev/stderr
    policy_url="$API_url/v1/autoscaler/apps/$app_guid/policy" 
    do_delete_policy  > /dev/stderr
    ((returncode+=$?))
    do_create_policy ${Resource_DIR}/file/policy/dynamic.json  > /dev/stderr
    ((returncode+=$?))

    echo " >>> The app $TestAPP_NAME will be scaled out according to policy:  "
    cat ${Resource_DIR}/file/policy/dynamic.json 

    echo 
    echo " >>> Now please add workload to $TestAPP_NAME Manually to trigger scaling." 
    echo " >>> Press Any key once the workload reaches a  proper value " 
    read y

    startTime=`date "+%H:%M"` 
     ((startTimestamp=$(date +%s)\*1000))
    echo " >>> The script below will detect application instance change for about 20 minutes." 
    checkInstance
    ((returncode+=$?))
    endTime=`date "+%H:%M"` 
     ((endTimestamp=$(date +%s)\*1000))

    setup_PublicAPI_TestConfig > /dev/stderr
    metrics_cf_url="$API_url/v1/autoscaler/apps/$app_guid/metrics?startTime=$startTimestamp&endTime=$endTimestamp"
    do_get_metric  > /dev/stderr  
    ((returncode+=$?))

    echo " >>> Query application metrics during $startTime ~ $endTime :  "
    cat $responseFile
    echo

    setup_PublicAPI_TestConfig > /dev/stderr
    history_cf_url="$API_url/v1/autoscaler/apps/$app_guid/scalinghistory?startTime=$startTimestamp&endTime=$endTimestamp"   
    do_get_scaling_history  > /dev/stderr    
    ((returncode+=$?))

    echo " >>> Query scaling history during $startTime ~ $endTime :  "
    cat $responseFile
    echo

    setup_PublicAPI_TestConfig > /dev/stderr
    do_delete_policy  > /dev/stderr
    cleanup_PublicAPI_TestConfig > /dev/stderr

    return $returncode
}


function setScheduleTime(){
    local fileName=$1
    if [ "$(uname)" == "Darwin" ]; then
        startDateValue=`date -u "+%Y-%m-%d"`
        endDateValue=`date -u "+%Y-%m-%d"`
        startTimeValue=`date -u "+%H:%M"` 
        endTimeValue=`date -u -v +2M "+%H:%M"`       
    elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
        startDateValue=`date -u "+%Y-%m-%d"`
        endDateValue=`date -u "+%Y-%m-%d"`
        startTimeValue=`date -u "+%H:%M"`
        endTimeValue=`date -d -u "2 minutes" "+%H:%M"`  
    fi

    if [ "$(uname)" == "Darwin" ]; then
            sed -i '' "s/{startDateValue}/$startDateValue/g" $fileName
            sed -i '' "s/{endDateValue}/$endDateValue/g" $fileName
            sed -i '' "s/{startTimeValue}/$startTimeValue/g" $fileName
            sed -i '' "s/{endTimeValue}/$endTimeValue/g" $fileName
    elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
            sed -i "s/{startDateValue}/$startDateValue/g" $fileName
            sed -i "s/{endDateValue}/$endDateValue/g" $fileName
            sed -i "s/{startTimeValue}/$startTimeValue/g" $fileName
            sed -i "s/{endTimeValue}/$endTimeValue/g" $fileName
    fi
        
}



function doCheckScaleByRecurringSchedule(){
    schedulePolicy=schedule.json
    cp ${Resource_DIR}/file/policy/recurringSchedule.json.template  schedule.json
    setScheduleTime $schedulePolicy

    cf scale $TestAPP_NAME -i 1 > /dev/stderr
    local returncode=0

    echo " >>> Test Scaling with Schedule: "
    setup_PublicAPI_TestConfig > /dev/stderr
    policy_url="$API_url/v1/autoscaler/apps/$app_guid/policy" 
    do_delete_policy  > /dev/stderr
    ((returncode+=$?))
    do_create_policy $schedulePolicy > /dev/stderr
    ((returncode+=$?))

    echo " >>> The app $TestAPP_NAME will be scaled out after 2 minutes according to policy:  "
    cat $schedulePolicy
    rm $schedulePolicy

    startTime=`date "+%H:%M"` 
    ((startTimestamp=$(date +%s)\*1000))
    checkInstance
    ((returncode+=$?))
    endTime=`date "+%H:%M"`  
    ((endTimestamp=$(date +%s)\*1000))
   

    setup_PublicAPI_TestConfig > /dev/stderr
    history_cf_url="$API_url/v1/autoscaler/apps/$app_guid/scalinghistory?startTime=$startTimestamp&endTime=$endTimestamp"   
    do_get_scaling_history  > /dev/stderr    
    ((returncode+=$?))

    echo " >>> Query scaling history during $startTime ~ $endTime : "
    cat $responseFile
    echo
  
    setup_PublicAPI_TestConfig > /dev/stderr
    do_delete_policy  > /dev/stderr
    ((returncode+=$?))

    cleanup_PublicAPI_TestConfig > /dev/stderr

    return $returncode


}








    


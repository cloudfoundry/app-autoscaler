#!/bin/bash

function do_login() {
    echo "============== Login CF ============================================"
    cf  login -a $domain -u $username -p $password -o $org -s $space --skip-ssl-validation
    if [ $? -ne 0 ];then
        cf create-org $org
        cf login -a $domain -u $username -p $password -o $org -s $space --skip-ssl-validation
        cf create-space $space
        cf login -a $domain -u $username -p $password -o $org -s $space --skip-ssl-validation
        if [ $? -ne 0 ]; then
            echo "Fail to login."
            return 254
        fi
    fi
    echo "============== Login CF successfully ============================================"

}


function do_cleanenv() {
   echo "=============== Clean test environment =========================================="
    cf service $SEVICE_INSTANCE_NAME
    if [ $? -eq 0 ]; then
        cf delete-service $SEVICE_INSTANCE_NAME -f
    fi 
    cf app $TestAPP_NAME
    if [ $? -eq 0 ]; then
            cf delete $TestAPP_NAME -f
    fi
    echo "=============== Clean test environment successfully =========================================="
}

function do_pushApp() {

    echo "============== Push App ============================================"
        cf app $TestAPP_NAME
        if [ $? -eq 0 ]; then
            cf delete $TestApp_
        fi
        cf push  $TestAPP_NAME -p $APP_FILE  --random-route 
        if [ $? -ne 0 ]; then
            echo "Fail to push application."
            return 254
        fi
    echo "============== Push App successfully================================="
}

function do_pushApp_nostart() {

    echo "============== Push App ============================================"
        cf app $TestAPP_NAME
        if [ $? -eq 0 ]; then
            cf delete $TestApp_
        fi
        cf push  $TestAPP_NAME -p $APP_FILE  --random-route --no-start
        if [ $? -ne 0 ]; then
            echo "Fail to push application."
            return 254
        fi
    echo "============== Push App successfully================================="
}

function setup_TestEnv() {
    do_login
    do_cleanenv
    do_pushApp_nostart
}




    


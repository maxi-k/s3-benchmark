#!/bin/bash

print_missing_parameters() {
    echo "Need to provide \"aws-settings\" bash file which sets at least required parameters!"
    echo "Required Parameters: AMI, IAM_PROFILE, AVAILABILITY_ZONE"
    echo "Optional Parameters: AVAILABILITY_ZONE, KEY_NAME"
    exit 1
}

print_configuration() {
    echo -e "+--------------- \033[1;32mRunning with Configuration:\033[0m--------------+"
    echo -e "| REGION:\t $REGION \033[59G|"
    echo -e "| AVAIL.ZONE:\t ${AVAILABILITY_ZONE:-unset} \033[59G|"
    echo -e "| AMI:\t\t $AMI\033[59G|"
    echo -e "| IAM_PROFILE:\t $IAM_PROFILE\033[59G|"
    echo -e "| KEY_NAME:\t ${KEY_NAME:-unset} \033[59G|"
    # echo -e "| SUBNET:\t $SUBNET\t\033[59G|"
    echo -e "+---------------------------------------------------------+\n\n"
}

ensure_parameters() {
    [ -f aws-settings ] && source ./aws-settings || print_missing_parameters
    REGION=${REGION:-"eu-central-1"}
}

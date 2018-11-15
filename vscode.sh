#!/bin/bash
# this script creates PROJ_GOPATH folder for the project in GOPATH

PROJ_ROOT=`pwd`
GOROOT=`go env GOROOT`
echo "Working in $PROJ_ROOT"

# include parse_yaml function
source .project/yaml.sh
create_variables ./config.yml

ORG_NAME=$project_org
PROJ_NAME=$project_name
REPO_NAME=$ORG_NAME/$PROJ_NAME
REL_PATH_TO_GOPATH=../../../..

echo "Repo: $REPO_NAME"

if [ -z "${ORG_NAME##*/*}" ] ;then
    #echo "'$ORG_NAME' contains: '/'."
    REL_PATH_TO_GOPATH=../../../..
else
    #echo "'$ORG_NAME' does not contain: '/'."
    REL_PATH_TO_GOPATH=../../..
fi

if [[ "$PWD" = *src/$REPO_NAME ]]; then
#
# Already in GOPATH format
#
pushd $REL_PATH_TO_GOPATH
CWD=`pwd`
PROJ_GOPATH_DIR=$CWD
PROJ_PACKAGE=$REPO_NAME
PROJ_GOPATH=$CWD
echo "PROJ_GOPATH=$PROJ_GOPATH"

export PROJ_DIR=$PROJ_ROOT
export PROJ_GOPATH_DIR="$PROJ_GOPATH_DIR"
export PROJ_GOPATH=$PROJ_GOPATH
export GOPATH=$PROJ_GOPATH
export GOROOT=$GOROOT
export PATH=$PATH:$PROJ_GOPATH/bin:$PROJ_DIR/bin:$PROJ_DIR/.tools/bin:$GOROOT/bin
env | grep GO
popd

code . & make devtools
else
#
# Not in GOPATH format
#
echo "WARNING: this project is not cloned in GOPATH"

ORG_NAME=$project_org
PROJ_NAME=$project_name
REPO_NAME=$ORG_NAME/$PROJ_NAME
export GOPATH=/tmp/gopath/$PROJ_NAME
export PROJ_GOPATH=$GOPATH
export PATH=$PATH:$PROJ_ROOT/.tools/bin

make gopath

echo "Opening in $GOPATH/src/$REPO_NAME"
code $GOPATH/src/$REPO_NAME & make devtools
fi

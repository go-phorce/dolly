#!/bin/bash
# this script creates PROJ_GOPATH folder for the project in GOPATH

ROOT=`pwd`
GOROOT=`go env GOROOT`
echo "Working in $ROOT"

# include parse_yaml function
source .project/yaml.sh
create_variables ./config.yml

ORG_NAME=$project_org
PROJ_NAME=$project_name
REPO_NAME=$ORG_NAME/$PROJ_NAME

echo "Repo: $REPO_NAME"

if [[ "$PWD" = *src/$REPO_NAME ]]; then
#
# Already in GOPATH format
#
pushd ../../../..
CWD=`pwd`
PROJ_GOPATH_DIR=$CWD
PROJ_PACKAGE=$REPO_NAME
PROJ_GOPATH=$CWD
echo "PROJ_GOPATH=$PROJ_GOPATH"

export PROJ_DIR=$ROOT
export PROJ_GOPATH_DIR="$PROJ_GOPATH_DIR"
export PROJ_GOPATH=$PROJ_GOPATH
export GOPATH=$PROJ_GOPATH
export GOROOT=$GOROOT
export PATH=$PATH:$PROJ_GOPATH/bin:$GOROOT/bin
env | grep GO
popd

code . & make devtools
else
#
# Not in GOPATH format
#
echo "WARNING: this project is not cloned in GOPATH"

pushd ..
CWD=`pwd`
PROJ_GOPATH_DIR=gopath
PROJ_PACKAGE=$REPO_NAME
PROJ_GOPATH=$CWD/$PROJ_GOPATH_DIR
echo "PROJ_GOPATH=$PROJ_GOPATH"

[ -d "$PROJ_GOPATH_DIR/src/$REPO_NAME" ] && rm -f "$PROJ_GOPATH_DIR/src/$REPO_NAME"
mkdir -p "$PROJ_GOPATH_DIR/src/$ORG_NAME"
ln -s ../../../../$PROJ_NAME "$PROJ_GOPATH_DIR/src/$REPO_NAME"

export PROJ_DIR=$ROOT
export PROJ_GOPATH_DIR="../$PROJ_GOPATH_DIR"
export PROJ_GOPATH=$PROJ_GOPATH
export GOPATH=$PROJ_GOPATH
export GOROOT=$GOROOT
export PATH=$PATH:$PROJ_GOPATH/bin:$PROJ_GOPATH/.tools/bin:$GOROOT/bin
env | grep GO
popd

code "$PROJ_GOPATH/src/$REPO_NAME" & make devtools
fi

#!/bin/bash

#
# config-softhsm.sh
#   --pin {pin}         - specify PIN
#   --pin-file {file}   - load or save generated PIN to file
#   --generate-pin      - generate PIN if does not exist
#   --slot {slot}       - slot name
#   --module {module}   - optional HSM module
#   --tokens-dir        - folder to store softhsm tokens
#   --out-cfg           - output file with configuration
#

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -p|--pin)
    HSM_PIN="$2"
    shift # past argument
    shift # past value
    ;;
    -f|--pin-file)
    HSM_PINFILE="$2"
    shift # past argument
    shift # past value
    ;;
    -s|--slot)
    HSM_SLOT="$2"
    shift # past argument
    shift # past value
    ;;
    -m|--module)
    HSM_MODULE="$2"
    shift # past argument
    shift # past value
    ;;
    -d|--tokens-dir)
    TOKEN_DIR="$2"
    shift # past argument
    shift # past value
    ;;
    -o|--out-cfg)
    CONFIG_FILE="$2"
    shift # past argument
    shift # past value
    ;;
    -g|--generate-pin)
    GENERATE_PIN=YES
    shift # past argument
    ;;
    --list-slots)
    LIST_SLOTS=YES
    shift # past argument
    ;;
    --list-object)
    LIST_OBJECTS=YES
    shift # past argument
    ;;
    --force)
    FORCE=YES
    shift # past argument
    ;;
    --delete)
    DELETE_TOKEN=YES
    shift # past argument
    ;;
    *)    # unknown option
    POSITIONAL+=("$1") # save it in an array for later
    shift # past argument
    ;;
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

[ -z "$HSM_SLOT" ] && echo "--slot is not provided" && exit 1

PKCS11_TOOL=`which pkcs11-tool`
SOFTHSM_TOOL=`which softhsm2-util`

[ -z "$PKCS11_TOOL" ] && echo "Please install pkcs11-tool" && exit 1
[ -z "$SOFTHSM_TOOL" ] && echo "Please install softhsm2" && exit 1

if [[ -z "$HSM_MODULE" ]]; then
UNAME=`uname`
echo "UNAME=$UNAME"

#
# Set path to HSM module depending on platform
#
if [ "$UNAME" = "Darwin" ]; then
  	# OSX Settings: Use `brew install engine_pkcs11 opensc libp11`
  	HSM_SPYMODULE=/usr/local/Cellar/opensc/0.17.0/lib/pkcs11-spy.so
  	HSM_MODULE=/usr/local/Cellar/softhsm/2.3.0/lib/softhsm/libsofthsm2.so
fi
    if [ "$UNAME" = "Linux" ]; then
        OS_REV=`uname -r`
        echo "OS_REV=$OS_REV"
        if [ -f "/usr/lib/softhsm/libsofthsm2.so" ] ;then
            HSM_MODULE=/usr/lib/softhsm/libsofthsm2.so
        elif [ -f "/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so" ] ;then
            HSM_MODULE=/usr/lib/x86_64-linux-gnu/softhsm/libsofthsm2.so
        else
            HSM_MODULE=/usr/lib64/pkcs11/libsofthsm2.so
        fi

        if [ -f "/usr/lib/x86_64-linux-gnu/pkcs11-spy.so" ] ;then
            HSM_SPYMODULE=/usr/lib/x86_64-linux-gnu/pkcs11-spy.so
        elif [ -f "/usr/lib/x86_64-linux-gnu/pkcs11-spy.so" ] ;then
            HSM_SPYMODULE=/usr/lib/x86_64-linux-gnu/pkcs11-spy.so
        else
            HSM_SPYMODULE=/usr/lib64/pkcs11/pkcs11-spy.so
        fi
    fi
fi

[ ! -f ${HSM_MODULE} ] && echo "HSM module not found: ${HSM_MODULE}" && exit 1

if [[ -z "$TOKEN_DIR" ]]; then
    TOKEN_DIR=~/softhsm2/tokens
fi

if [[ -z "$HSM_PIN" && -f $HSM_PINFILE ]]; then
    HSM_PIN_VAL=`cat $HSM_PINFILE`
    HSM_PIN="file:${HSM_PINFILE}"
fi

if [[ -z "$HSM_PIN" && "$GENERATE_PIN" == "YES" ]]; then
    if [[ ! -z "$HSM_PINFILE" ]]; then
        HSM_PIN_VAL=`echo $RANDOM$RANDOM$RANDOM`
        echo $HSM_PIN_VAL > $HSM_PINFILE
        HSM_PIN="file:${HSM_PINFILE}"
    else
        HSM_PIN_VAL=`echo $RANDOM$RANDOM$RANDOM`
        HSM_PIN=$HSM_PIN_VAL
    fi
fi

[ -z "$HSM_PIN" ] && echo "pin is not provided, use --pin | --pin-file | --generate-pin" && exit 1

echo HSM_PIN     = "${HSM_PIN}"
echo HSM_SLOT    = "${HSM_SLOT}"
echo HSM_MODULE  = "${HSM_MODULE}"
echo PKCS11_TOOL = "${PKCS11_TOOL}"
echo SOFTHSM_TOOL= "${SOFTHSM_TOOL}"
echo TOKEN_DIR   = "${TOKEN_DIR}"

if [[ "$FORCE" == "YES" ]]; then
    rm -rf ~/.config/softhsm2
fi

if [[ ! -f ~/.config/softhsm2/softhsm2.conf ]]; then
    echo 'Creating ~/.config/softhsm2/softhsm2.conf'
    mkdir -p ${TOKEN_DIR}
    mkdir -p ~/.config/softhsm2
    echo "directories.tokendir = $TOKEN_DIR" > ~/.config/softhsm2/softhsm2.conf
fi

if [[ "$DELETE_TOKEN" == "YES" ]]; then
    softhsm2-util --show-slots | grep -q "${HSM_SLOT}" && softhsm2-util --delete-token --token="${HSM_SLOT}" || echo "${HSM_SLOT} does not exist"
    echo "*** Creating ${HSM_SLOT} slot"
fi

# create slot if it does not exist
softhsm2-util --show-slots | grep -q "${HSM_SLOT}" || softhsm2-util --init-token --free --label "${HSM_SLOT}" --force --pin ${HSM_PIN_VAL} --so-pin so${HSM_PIN_VAL}

[[ ! -z "$CONFIG_FILE" ]] && echo { \"Manufacturer\" : \"SoftHSM\", \"Path\": \"$HSM_MODULE\", \"TokenLabel\": \"$HSM_SLOT\", \"Pin\": \"$HSM_PIN\" } > $CONFIG_FILE

echo "HSM_PIN_VAL=${HSM_PIN_VAL}"
cat $CONFIG_FILE

if [[ "$LIST_SLOTS" == "YES" ]]; then
    echo "*** Slots:"
    pkcs11-tool --module "$HSM_MODULE" --list-slots
    #softhsm2-util --show-slots
fi
if [[ "$LIST_OBJECTS" == "YES" ]]; then
    echo "*** Objects:"
    pkcs11-tool --module "$HSM_MODULE" --login --pin $HSM_PIN_VAL --token-label "${HSM_SLOT}" --list-object
fi

#!/bin/bash

#
# gen_test_certs.sh
#   --out-dir {dir}     - specifies output folder
#   --csr-dir {dir}     - specifies folder with CSR templates
#   --prefix {prefix}   - specifies prefix for files, by default: ${PREFIX}
#   --ca-config {file}  - specifies CA configuration file
#   --ca-bundle {file}  - specifies root CA bundle file
#   --root-cert {cert}  - specifies root CA certificate
#   --root-key {key}    - specifies root CA key
#   --ca1-cert {cert}   - specifies Level 1 CA certificate
#   --ca1-key {key}     - specifies Level 1 CA key
#   --ca2-cert {cert}   - specifies Level 2 CA certificate
#   --ca2-key {key}     - specifies Level 2 CA key
#   --root              - specifies if Root CA certificate and key should be generated
#   --ca1               - specifies if Level 1 CA certificate and key should be generated
#   --ca2               - specifies if Level 2 CA certificate and key should be generated
#   --server            - specifies if server TLS certificate and key should be generated
#   --client            - specifies if client certificate and key should be generated
#   --peers             - specifies if peers certificate and key should be generated
#   --bundle            - specifies if Int CA Bundle should be created
#   --force             - specifies to force issuing the cert even if it exists
#

POSITIONAL=()
while [[ $# -gt 0 ]]
do
key="$1"

case $key in
    -o|--out-dir)
    OUT_DIR="$2"
    shift # past argument
    shift # past value
    ;;
    -o|--csr-dir)
    CSR_DIR="$2"
    shift # past argument
    shift # past value
    ;;
    -p|--prefix)
    PREFIX="$2"
    shift # past argument
    shift # past value
    ;;
    -c|--ca-config)
    CA_CONFIG="$2"
    shift # past argument
    shift # past value
    ;;
    -b|--ca-bundle)
    CA_BUNDLE="$2"
    shift # past argument
    shift # past value
    ;;
    --root-cert)
    ROOT_CA_CERT="$2"
    shift # past argument
    shift # past value
    ;;
    --root-key)
    ROOT_CA_KEY="$2"
    shift # past argument
    shift # past value
    ;;
    --ca1-cert)
    CA1_CERT="$2"
    shift # past argument
    shift # past value
    ;;
    --ca1-key)
    CA1_KEY="$2"
    shift # past argument
    shift # past value
    ;;
    --ca2-cert)
    CA2_CERT="$2"
    shift # past argument
    shift # past value
    ;;
    --ca2-key)
    CA2_KEY="$2"
    shift # past argument
    shift # past value
    ;;
    --root)
    ROOTCA=YES
    shift # past argument
    ;;
    --ca1)
    CA1=YES
    shift # past argument
    ;;
    --ca2)
    CA2=YES
    shift # past argument
    ;;
    --server)
    SERVER=YES
    shift # past argument
    ;;
    --admin)
    ADMIN=YES
    shift # past argument
    ;;
    --client)
    CLIENT=YES
    shift # past argument
    ;;
    --peers)
    PEERS=YES
    shift # past argument
    ;;
    --force)
    FORCE=YES
    shift # past argument
    ;;
    --bundle)
    BUNDLE=YES
    shift # past argument
    ;;
    *)
    echo "invalid flag $key: use --help to see the option"
    exit 1
esac
done
set -- "${POSITIONAL[@]}" # restore positional parameters

[ -z "$OUT_DIR" ] &&  echo "Specify --out-dir" && exit 1
[ -z "$CSR_DIR" ] &&  echo "Specify --csr-dir" && exit 1
[ -z "$CA_CONFIG" ] && echo "Specify --ca-config" && exit 1
[ -z "$PREFIX" ] && PREFIX=test_
[ -z "$ROOT_CA_CERT" ] && ROOT_CA_CERT=${OUT_DIR}/${PREFIX}root_CA.pem
[ -z "$ROOT_CA_KEY" ] && ROOT_CA_KEY=${OUT_DIR}/${PREFIX}root_CA-key.pem
[ -z "$CA1_CERT" ] && CA1_CERT=${OUT_DIR}/${PREFIX}issuer1_CA.pem
[ -z "$CA1_KEY" ] && CA1_KEY=${OUT_DIR}/${PREFIX}issuer1_CA-key.pem
[ -z "$CA2_CERT" ] && CA2_CERT=${OUT_DIR}/${PREFIX}issuer2_CA.pem
[ -z "$CA2_KEY" ] && CA2_KEY=${OUT_DIR}/${PREFIX}issuer2_CA-key.pem
[ -z "$CA_BUNDLE" ] && CA_BUNDLE=${OUT_DIR}/${PREFIX}cabundle.pem

HOSTNAME=`hostname`

echo "OUT_DIR      = ${OUT_DIR}"
echo "CSR_DIR      = ${CSR_DIR}"
echo "CA_CONFIG    = ${CA_CONFIG}"
echo "CA_BUNDLE    = ${CA_BUNDLE}"
echo "PREFIX       = ${PREFIX}"
echo "BUNDLE       = ${BUNDLE}"
echo "FORCE        = ${FORCE}"
echo "ROOT_CA_CERT = $ROOT_CA_CERT"
echo "ROOT_CA_KEY  = $ROOT_CA_KEY"
echo "CA1_CERT     = $CA1_CERT"
echo "CA1_KEY      = $CA1_KEY"
echo "CA2_CERT     = $CA2_CERT"
echo "CA2_KEY      = $CA2_KEY"

if [[ "$ROOTCA" == "YES" && ("$FORCE" == "YES" || ! -f ${ROOT_CA_KEY}) ]]; then echo "*** generating ${ROOT_CA_CERT/.pem/''}"
    cfssl genkey -initca \
        -config=${CA_CONFIG} \
        ${CSR_DIR}/${PREFIX}root_CA.json | cfssljson -bare ${ROOT_CA_CERT/.pem/''}
fi

if [[ "$CA1" == "YES" && ("$FORCE" == "YES" || ! -f ${CA1_KEY}) ]]; then
    echo "*** generating CA1 cert: ${CA1_CERT}"
    cfssl genkey -initca \
        -config=${CA_CONFIG} \
        ${CSR_DIR}/${PREFIX}issuer1_CA.json | cfssljson -bare ${CA1_CERT/.pem/''}

    cfssl sign \
        -config=${CA_CONFIG} \
        -profile=L1_CA \
        -ca ${ROOT_CA_CERT} \
        -ca-key ${ROOT_CA_KEY} \
        -csr ${CA1_CERT/.pem/.csr} | cfssljson -bare ${CA1_CERT/.pem/''}
fi

if [[ "$CA2" == "YES" && ("$FORCE" == "YES" || ! -f ${CA2_KEY}) ]]; then
    echo "*** generating CA2 cert:  ${CA2_CERT}"
    cfssl genkey -initca \
        -config=${CA_CONFIG} \
        ${CSR_DIR}/${PREFIX}issuer2_CA.json | cfssljson -bare ${CA2_CERT/.pem/''}

    cfssl sign \
        -config=${CA_CONFIG} \
        -profile=L2_CA \
        -ca ${CA1_CERT} \
        -ca-key ${CA1_KEY} \
        -csr ${CA2_CERT/.pem/.csr} | cfssljson -bare ${CA2_CERT/.pem/''}
fi

if [[ "$BUNDLE" == "YES" && ("$FORCE" == "YES" || ! -f ${CA_BUNDLE}) ]]; then
    echo "*** CA bundle: ${CA_BUNDLE}"
    cat ${CA1_CERT} > ${CA_BUNDLE}
    cat ${CA2_CERT} >> ${CA_BUNDLE}
fi

if [[ "$ADMIN" == "YES" && ("$FORCE" == "YES" || ! -f ${OUT_DIR}/${PREFIX}admin-key.pem) ]]; then
    echo "*** generating admin cert: ${OUT_DIR}/${PREFIX}admin"
    cfssl gencert \
        -config=${CA_CONFIG} \
        -profile=client \
        -ca ${CA2_CERT} \
        -ca-key ${CA2_KEY} \
        ${CSR_DIR}/${PREFIX}admin.json | cfssljson -bare ${OUT_DIR}/${PREFIX}admin
        cat ${CA_BUNDLE} >> ${OUT_DIR}/${PREFIX}admin.pem
fi

if [[ "$SERVER" == "YES" && ("$FORCE" == "YES" || ! -f ${OUT_DIR}/${PREFIX}server-key.pem) ]]; then
    echo "*** generating server cert: ${OUT_DIR}/${PREFIX}server"
    cfssl gencert \
        -config=${CA_CONFIG} \
        -profile=server \
        -ca ${CA2_CERT} \
        -ca-key ${CA2_KEY} \
        -hostname=localhost,127.0.0.1,${HOSTNAME} \
        ${CSR_DIR}/${PREFIX}server.json | cfssljson -bare ${OUT_DIR}/${PREFIX}server
        cat ${CA_BUNDLE} >> ${OUT_DIR}/${PREFIX}server.pem
fi

if [[ "$CLIENT" == "YES" && ("$FORCE" == "YES" || ! -f ${OUT_DIR}/${PREFIX}client-key.pem) ]]; then
    echo "*** generating client cert: ${OUT_DIR}/${PREFIX}client"
    cfssl gencert \
        -config=${CA_CONFIG} \
        -profile=client \
        -ca ${CA2_CERT} \
        -ca-key ${CA2_KEY} \
        ${CSR_DIR}/${PREFIX}client.json | cfssljson -bare ${OUT_DIR}/${PREFIX}client
        cat ${CA_BUNDLE} >> ${OUT_DIR}/${PREFIX}client.pem
fi

if [[ "$PEERS" == "YES" && ("$FORCE" == "YES" || ! -f ${OUT_DIR}/${PREFIX}peers-key.pem) ]]; then
    echo "*** generating peers cert: ${OUT_DIR}/${PREFIX}peers"
    cfssl gencert \
        -config=${CA_CONFIG} \
        -profile=peer \
        -ca ${CA2_CERT} \
        -ca-key ${CA2_KEY} \
        -hostname=localhost,127.0.0.1,${HOSTNAME} \
        ${CSR_DIR}/${PREFIX}peers.json | cfssljson -bare ${OUT_DIR}/${PREFIX}peers
        cat ${CA_BUNDLE} >> ${OUT_DIR}/${PREFIX}peers.pem
fi

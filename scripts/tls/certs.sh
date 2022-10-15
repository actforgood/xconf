#!/usr/bin/env bash

#
# This script generates TLS (CA and server) certs for testing purposes.
#
# Example of usage of this script:
# ./path/to/scripts/tls/certs.sh
#

SCRIPT_PATH=$(dirname "$(readlink -f "$0")")
CERTS_PATH="${SCRIPT_PATH}/certs"
mkdir -p "${CERTS_PATH}"

# Create CA cert
openssl req -x509                                           \
  -newkey rsa:4096                                          \
  -nodes                                                    \
  -days 365                                                 \
  -keyout "${CERTS_PATH}/ca_key.pem"                        \
  -out "${CERTS_PATH}/ca_cert.pem"                          \
  -subj /C=US/ST=CA/L=SiliconValley/O=xconf/CN=test-ca/     \
  -config "${SCRIPT_PATH}/openssl.cnf"                      \
  -extensions test_ca                                       \
  -sha256

# Generate Etcd Server cert
openssl genrsa -out "${CERTS_PATH}/etcd_server_key.pem" 4096

openssl req -new                                                    \
  -key "${CERTS_PATH}/etcd_server_key.pem"                          \
  -days 365                                                         \
  -out "${CERTS_PATH}/etcd_server_csr.pem"                          \
  -subj /C=US/ST=CA/L=SiliconValley/O=xconf/CN=xconf-etcds/    \
  -config "${SCRIPT_PATH}/openssl.cnf"                              \
  -reqexts test_etcd_server

openssl x509 -req                           \
  -in "${CERTS_PATH}/etcd_server_csr.pem"   \
  -CAkey "${CERTS_PATH}/ca_key.pem"         \
  -CA "${CERTS_PATH}/ca_cert.pem"           \
  -days 365                                 \
  -set_serial 1000                          \
  -out "${CERTS_PATH}/etcd_server_cert.pem" \
  -extfile "${SCRIPT_PATH}/openssl.cnf"     \
  -extensions test_etcd_server              \
  -sha256

openssl verify -verbose -CAfile "${CERTS_PATH}/ca_cert.pem"  "${CERTS_PATH}/etcd_server_cert.pem"

rm -rf "${CERTS_PATH}"/*_csr.pem

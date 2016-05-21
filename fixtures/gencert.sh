 #!/bin/bash
set -e

# This relies on certstrap - https://github.com/square/certstrap

depot=$(mktemp -d)
certstrap --depot-path ${depot} init --common-name "Test CA" --years 20 --passphrase ""
certstrap --depot-path ${depot} request-cert --common-name test-server --ip 127.0.0.1 --passphrase ""
certstrap --depot-path ${depot} sign test-server --CA 'Test CA' --years 10

mkdir -p certs
cat ${depot}/test-server.crt ${depot}/Test_CA.crt > certs/server.pem
cp ${depot}/test-server.key certs/server.key

rm -rf ${depot}

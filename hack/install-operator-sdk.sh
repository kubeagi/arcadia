#!/bin/bash -e

OPERATOR_SDK_VERSION="v1.24.1"
ARCH=$(go env GOARCH)
OS=$(go env GOOS)
URL="https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_SDK_VERSION}/operator-sdk_${OS}_${ARCH}"

operator_sdk_version=$(operator-sdk version)

if echo $operator_sdk_version | grep -q "$OPERATOR_SDK_VERSION"; then
    echo "operator-sdk version ${OPERATOR_SDK_VERSION} found, exiting..."
    exit 0 
fi 

echo "Installing operator-sdk version ${OPERATOR_SDK_VERSION} to /usr/local/bin/operator-sdk"
curl -L $URL > operator-sdk
chmod +x operator-sdk
sudo mv operator-sdk /usr/local/bin
operator-sdk version

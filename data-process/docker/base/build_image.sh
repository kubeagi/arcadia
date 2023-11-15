set e

release_image="python:3.10.13"


docker build -f ./Dockerfile.base -t ${release_image} --build-arg GIT_VERSION="$gitVersion" .
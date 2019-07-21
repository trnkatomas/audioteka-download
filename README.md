# Audioteka download
Simplistic tool for downloading the latest audio book from audioteka.cz


Cross compilation for RaspberryPi
```{bash}
docker run -it --rm \                             
-v "$PWD":/go/src/myrepo/mypackage -w /go/src/myrepo/mypackage \
-e GOOS=linux -e GOARCH=arm -e CGO_ENABLED=1 \
-e CC=arm-linux-gnueabihf-gcc anakros/goarm /bin/bash -c "go get . && go build -o binary-armhf-linux"
```
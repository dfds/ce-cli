#!/bin/sh

build_for_platform()
{
  GOOS=$1 GOARCH=$2 go build -o builds/ce-$1-$2
}

mkdir -p builds

build_for_platform linux arm64
build_for_platform linux amd64
build_for_platform darwin arm64
build_for_platform darwin amd64
build_for_platform windows amd64
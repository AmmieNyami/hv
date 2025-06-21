#!/bin/sh

CC=musl-gcc CGO_ENABLED=1 go build \
            -tags "sqlite_omit_load_extension" \
            -ldflags '-linkmode external -extldflags "-static"'

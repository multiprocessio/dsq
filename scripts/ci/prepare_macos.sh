#!/usr/bin/env bash

set -eux

sudo curl -LO https://go.dev/dl/go1.18.darwin-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.18.darwin-amd64.tar.gz
sudo mv /usr/local/go/bin/go /usr/local/bin/go
sudo mv /usr/local/go/bin/gofmt /usr/local/bin/gofmt

#!/usr/bin/env bash

set -eux

sudo curl -LO https://go.dev/dl/go1.18.darwin-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.18.darwin-amd64.tar.gz
sudo mv /usr/local/go/bin/go /usr/local/bin/go
sudo mv /usr/local/go/bin/gofmt /usr/local/bin/gofmt

# Install 7z
curl -LO https://github.com/jinfeihan57/p7zip/releases/download/v17.04/macos-10.15-p7zip.zip
unzip macos-10.15-p7zip.zip
./7z e testdata/taxi.csv.7z

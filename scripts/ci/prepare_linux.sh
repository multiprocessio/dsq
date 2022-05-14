#!/usr/bin/env bash

set -ex

sudo curl -LO https://go.dev/dl/go1.18.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.18.linux-amd64.tar.gz
sudo ln -s /usr/local/go/bin/go /usr/local/bin/go
sudo ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt

sudo apt install -y p7zip
7z e testdata/taxi.csv.7z

iwr -useb 'https://raw.githubusercontent.com/scoopinstaller/install/master/install.ps1' -outfile 'install.ps1'
.\install.ps1 -RunAsAdmin
Join-Path (Resolve-Path ~).Path "scoop\shims" >> $Env:GITHUB_PATH
scoop install jq zip curl

curl -L -O "https://go.dev/dl/go1.18.windows-amd64.zip"
unzip go1.18.windows-amd64.zip
Join-Path $pwd "go\bin" >> $Env:GITHUB_PATH
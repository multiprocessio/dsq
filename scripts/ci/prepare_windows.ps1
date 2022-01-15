Invoke-Expression (New-Object System.Net.WebClient).DownloadString('https://get.scoop.sh')
Join-Path (Resolve-Path ~).Path "scoop\shims" >> $Env:GITHUB_PATH
scoop install go jq zip
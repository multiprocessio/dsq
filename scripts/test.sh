#!/usr/bin/env bash

set -e

types="csv tsv parquet json jsonl xlsx ods"

for t in $types; do
    echo "Testing $t (pipe)."
    sqlcount="$(cat ./testdata/userdata.$t | ./dsq -s $t 'SELECT COUNT(1) AS c FROM {}' | jq '.[0].c')"
    if [[ "$sqlcount" != "1000" ]]; then
	echo "Bad SQL count for $t (pipe). Expected 1000, got $sqlcount."
	exit 1
    else
	echo "Pipe $t test successful."
    fi

    echo "Testing $t (file)."
    sqlcount="$(./dsq ./testdata/userdata.$t 'SELECT COUNT(1) AS c FROM {}' | jq '.[0].c')"
    if [[ "$sqlcount" != "1000" ]]; then
	echo "Bad SQL count for $t (file). Expected 1000, got $sqlcount."
	exit 1
    else
	echo "File $t test successful."
    fi
done

joined="$(./dsq testdata/join/users.csv testdata/join/ages.json "select {0}.name, {1}.age from {0} join {1} on {0}.id = {1}.id")"
expected='[{"age":88,"name":"Ted"}
,
{"age":56,"name":"Marjory"}
,
{"age":33,"name":"Micah"}
]'
if ! [[ "$joined" == "$expected" ]]; then
    echo "Bad join:"
    diff <(echo "$joined") <(echo "$expected")
    exit 1
fi

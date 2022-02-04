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

function test() {
    name="$1"
    expected="$2"
    got="$3"

    if ! [[ "$expected" == "$got" ]]; then
	printf "[FAIL] %s\n" "$name"
	diff <(echo "$got") <(echo "$expected")
	exit 1
    fi

    printf "[SUCCESS] %s\n" "$name"
}


# Join test

joined="$(./dsq testdata/join/users.csv testdata/join/ages.json "select {0}.name, {1}.age from {0} join {1} on {0}.id = {1}.id")"
expected='[{"age":88,"name":"Ted"},
{"age":56,"name":"Marjory"},
{"age":33,"name":"Micah"}]'

test "Join two file-tables" "$expected" "$joined"

# Nested values test

got=`./dsq ./testdata/nested/nested.json 'select name, "location.city" city, "location.address.number" address_number from {}'`
expected='[{"address_number":1002,"city":"Toronto","name":"Agarrah"},
{"address_number":19,"city":"Mexico City","name":"Minoara"},
{"address_number":12,"city":"New London","name":"Fontoon"}]'

test "Extract nested values" "$expected" "$got"

# No input test

expected="No input files."
got="$(./dsq 2>&1 || true)"

test "Handles no arguments correctly" "$expected" "$got"

# Not an array of data test

expected="Input is not an array of objects: testdata/bad/not_an_array.json."
got="$(./dsq testdata/bad/not_an_array.json 2>&1 || true)"

test "Does not allow querying on non-array data" "$expected" "$got"

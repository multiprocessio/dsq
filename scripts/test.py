#!/usr/bin/env python3

import os
import shlex
import subprocess
import sys

WIN = os.name == 'nt'

SHELL = 'system'
for i, a in enumerate(sys.argv):
    if a == '--shell':
        SHELL = sys.argv[i+1]


def cmd(to_run, s=SHELL):
    if s == 'system':
        return subprocess.check_output(shlex.split(to_run), stderr=subprocess.STDOUT)
    elif s == 'bash':
        return subprocess.check_output(['bash', '-c', to_run], stderr=subprocess.STDOUT)
    elif s == 'powershell':
        return subprocess.check_output(['powershell', '-Command', to_run], stderr=subprocess.STDOUT)
    elif s == 'cmd':
        return subprocess.check_output(['cmd.exe', '/C', to_run], stderr=subprocess.STDOUT)

    print('Unknown shell: ' + s)
    sys.exit(1)


tests = 0
failures = 0

def test(name, to_run, want, s=SHELL, fail=False):
    global tests
    global failures
    tests += 1
    skipped = True

    if WIN:
        to_run = to_run.replace('./dsq', './dsq.exe').replace('/', '\\')
    if s in ['cmd', 'powershell']:
        # Bash and powershell require nested quotes to be escaped
        to_run = to_run.replace('"', '\\"')

    try:
        got = cmd(to_run, s).decode()
    except Exception as e:
        if not fail:
            print(f'[FAILURE] ' + name + ', unexpected failure')
            print(e)
            failures += 1
            return
        else:
            got = e.output.decode()
            skipped = False
    if fail and skipped:
        print(f'[FAILURE] ' + name + ', unexpected success')
        failures += 1
        return
    if want.strip() != got.strip():
        print(f'[FAILURE] ' + name)
        try:
            print(cmd(f'diff <(echo "{want.strip()}") <(echo "{got.strip()}") || true', 'bash').decode())
        except Exception as e:
            print(e)
        failures += 1
        return

    print(f'[SUCCESS] ' + name)


types = ['csv', 'tsv', 'parquet', 'json', 'jsonl', 'xlsx', 'ods']
for t in types:
    if WIN:
        continue
    to_run = f"cat ./testdata/userdata.{t} | ./dsq -s {t} 'SELECT COUNT(1) AS c FROM {{}}' | jq '.[0].c'"
    test('SQL count for ' + t + ' pipe in bash', to_run, '1000', 'bash')

    to_run = f"./dsq ./testdata/userdata.{t} 'SELECT COUNT(1) AS c FROM {{}}' | jq '.[0].c'"
    test('SQL count for ' + t + ' file in bash', to_run, '1000', 'bash')

# Join test
to_run = "./dsq testdata/join/users.csv testdata/join/ages.json 'select {0}.name, {1}.age from {0} join {1} on {0}.id = {1}.id'"
want = """[{"age":88,"name":"Ted"},
{"age":56,"name":"Marjory"},
{"age":33,"name":"Micah"}]"""
test("Join two file-tables", to_run, want)

# Nested values test
to_run = """./dsq ./testdata/nested/nested.json 'select name, "location.city" city, "location.address.number" address_number from {}'"""
want = """[{"address_number":1002,"city":"Toronto","name":"Agarrah"},
{"address_number":19,"city":"Mexico City","name":"Minoara"},
{"address_number":12,"city":"New London","name":"Fontoon"}]"""
test("Extract nested values", to_run, want)

# No input test
to_run = "./dsq"
want = "No input files."
test("Handles no arguments correctly", to_run, want, fail=True)

# Not an array of data test
to_run = "./dsq ./testdata/bad/not_an_array.json"
want = "Input is not an array of objects: ./testdata/bad/not_an_array.json."
test("Does not allow querying on non-array data", to_run, want, fail=True)

# END OF TESTS

print(f"\n{tests - failures} of {tests} succeeded.\n")
if failures > 0:
    sys.exit(1)

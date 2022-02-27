#!/usr/bin/env python3

import os
import shlex
import subprocess
import sys
import tempfile

WIN = os.name == 'nt'

def cmd(to_run, bash=False):
    pieces = shlex.split(to_run)
    if WIN:
        for i, piece in enumerate(pieces):
            pieces[i] = piece.replace('./dsq', './dsq.exe').replace('/', '\\')
    elif bash or '|' in pieces:
        pieces = ['bash', '-c', to_run]

    return subprocess.check_output(pieces, stderr=subprocess.STDOUT, cwd=os.getcwd())

tests = 0
failures = 0

def test(name, to_run, want, fail=False):
    global tests
    global failures
    tests += 1
    skipped = True

    print('STARTING: ' + name)

    try:
        got = cmd(to_run).decode()
    except Exception as e:
        if not fail:
            print(f'  FAILURE: unexpected failure: ' + (e.output.decode() if isinstance(e, bytes) else str(e)))
            failures += 1
            print()
            return
        else:
            got = e.output.decode()
            skipped = False
    if fail and skipped:
        print(f'  FAILURE: unexpected success')
        failures += 1
        print()
        return
    if WIN and '/' in want:
        want = want.replace('/', '\\')
    if want.strip() != got.strip():
        print(f'  FAILURE')
        try:
            with tempfile.NamedTemporaryFile() as want_fp:
                want_fp.write(want.strip().encode())
                want_fp.flush()
                with tempfile.NamedTemporaryFile() as got_fp:
                    got_fp.write(got.strip().encode())
                    got_fp.flush()
                    print(cmd(f'diff {want_fp.name} {got_fp.name} || true', bash=True).decode())
        except Exception as e:
            print(e.cmd, e.output.decode())
        failures += 1
        print()
        return

    print(f'  SUCCESS\n')


types = ['csv', 'tsv', 'parquet', 'json', 'jsonl', 'xlsx', 'ods']
for t in types:
    if WIN:
        continue
    to_run = f"cat ./testdata/userdata.{t} | ./dsq -s {t} 'SELECT COUNT(1) AS c FROM {{}}' | jq '.[0].c'"
    test('SQL count for ' + t + ' pipe', to_run, '1000')

    to_run = f"./dsq ./testdata/userdata.{t} 'SELECT COUNT(1) AS c FROM {{}}' | jq '.[0].c'"
    test('SQL count for ' + t + ' file', to_run, '1000')


# No input test
to_run = "./dsq"
want = "No input files."
test("Handles no arguments correctly", to_run, want, fail=True)

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

# Not an array of data test
to_run = "./dsq ./testdata/bad/not_an_array.json"
want = "Input is not an array of objects: ./testdata/bad/not_an_array.json."
test("Does not allow querying on non-array data", to_run, want, fail=True)

# REGEXP support
to_run = """./dsq ./testdata/nested/nested.json "SELECT * FROM {} WHERE name REGEXP 'A.*'" """
want = '[{"location.address.number":1002,"location.city":"Toronto","name":"Agarrah"}]'
test("Supports filtering with REGEXP", to_run, want)

# Table aliases
to_run = """./dsq ./testdata/nested/nested.json "SELECT * FROM {} u WHERE u.name REGEXP 'A.*'" """
want = '[{"location.address.number":1002,"location.city":"Toronto","name":"Agarrah"}]'
test("Supports table aliases", to_run, want)

# END OF TESTS

print(f"{tests - failures} of {tests} succeeded.")
if failures > 0:
    sys.exit(1)

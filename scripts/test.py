#!/usr/bin/env python3

import glob
import json
import os
import shlex
import subprocess
import sys
import tempfile
from datetime import datetime

DEBUG = '-d' in sys.argv or '--debug' in sys.argv
WIN = os.name == 'nt'

def cmd(to_run, bash=False, doNotReplaceWin=False):
    pieces = shlex.split(to_run)
    if WIN and not doNotReplaceWin:
        for i, piece in enumerate(pieces):
            pieces[i] = piece.replace('./dsq', './dsq.exe').replace('/', '\\')
    elif bash or '|' in pieces:
        pieces = ['bash', '-c', to_run]

    return subprocess.run(pieces, cwd=os.getcwd(), capture_output=True, check=True)

tests = 0
failures = 0

def test(name, to_run, want, fail=False, sort=False, winSkip=False, within_seconds=None, want_stderr=None):
    global tests
    global failures
    
    skip = False
    for i, arg in enumerate(sys.argv):
        if arg == '-f' or arg == '--filter':
            if sys.argv[i+1].lower() not in name.lower():
                return
        if arg == '-fo' or arg == '--filter-out':
            if sys.argv[i+1].lower() in name.lower():
                return

    tests += 1
    skipped = True

    t1 = datetime.now()

    print('STARTING: ' + name)
    if DEBUG:
        print(to_run)

    if WIN and winSkip or skip:
      print('  SKIPPED\n')
      print()
      return

    try:
        res = cmd(to_run)
        got = res.stdout.decode()

        got_err = res.stderr.decode()
        if want_stderr and got_err != want_stderr:
            failures += 1
            print(f'  FAILURE: stderr mismatch. Got "{got_err}", wanted "{want_stderr}".')
            print()
            return
    
        if sort:
            got = json.dumps(json.loads(got), sort_keys=True)
            want = json.dumps(json.loads(want), sort_keys=True)
    except json.JSONDecodeError as e:
        failures += 1
        print('  FAILURE: bad JSON: ' + got)
        print()
        return
    except Exception as e:
        if not fail:
            print(f'  FAILURE: unexpected failure: {0} {1}', str(e), e.output.decode())
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
                    diff_res = cmd(f'diff {want_fp.name} {got_fp.name} || true', bash=True)
                    print(diff_res.stdout.decode())
        except Exception as e:
            print(e.cmd, e.output.decode())
        failures += 1
        print()
        return

    t2 = datetime.now()
    s = (t2-t1).seconds
    if within_seconds and s > within_seconds:
        print(f'  FAILURE: completed in {s} seconds. Wanted <{within_seconds}s')
        failures += 1
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
want_stderr = "No input files.\n"
test("Handles no arguments correctly", to_run, want="", want_stderr=want_stderr, fail=True)

# Join test
to_run = "./dsq testdata/join/users.csv testdata/join/ages.json 'select {0}.name, {1}.age from {0} join {1} on {0}.id = {1}.id'"
want = """[{"age":88,"name":"Ted"},
{"age":56,"name":"Marjory"},
{"age":33,"name":"Micah"}]"""
test("Join two file-tables", to_run, want, sort=True)

# Nested values test
to_run = """./dsq ./testdata/nested/nested.json 'select name, "location.city" city, "location.address.number" address_number from {}'"""
want = """[{"address_number":1002,"city":"Toronto","name":"Agarrah"},
{"address_number":19,"city":"Mexico City","name":"Minoara"},
{"address_number":12,"city":"New London","name":"Fontoon"}]"""
test("Extract nested values", to_run, want, sort=True)

# Not an array of data test
to_run = "./dsq ./testdata/bad/not_an_array.json 'SELECT * FROM {}'"
want_stderr = "Input is not an array of objects: ./testdata/bad/not_an_array.json.\n"
test("Does not allow querying on non-array data", to_run, want="", want_stderr=want_stderr, fail=True)

# REGEXP support
to_run = """./dsq ./testdata/nested/nested.json "SELECT * FROM {} WHERE name REGEXP 'A.*'" """
want = '[{"location.address.number":1002,"location.city":"Toronto","name":"Agarrah"}]'
test("Supports filtering with REGEXP", to_run, want, sort=True)

# Table aliases
to_run = """./dsq ./testdata/nested/nested.json "SELECT * FROM {} u WHERE u.name REGEXP 'A.*'" """
want = '[{"location.address.number":1002,"location.city":"Toronto","name":"Agarrah"}]'
test("Supports table aliases", to_run, want, sort=True)

# With path
to_run = """./dsq ./testdata/path/path.json "SELECT * FROM {0, 'data.data'} ORDER BY id DESC" """
want = '[{"id":3,"name":"Minh"},{"id":1,"name":"Corah"}]'
test("Supports path specification", to_run, want, sort=True)

# With path shorthand
to_run = """./dsq ./testdata/path/path.json "SELECT * FROM {'data.data'} ORDER BY id DESC" """
want = '[{"id":3,"name":"Minh"},{"id":1,"name":"Corah"}]'
test("Supports path specification shorthand", to_run, want, sort=True)

# Excel multiple sheets
to_run = """./dsq testdata/excel/multiple-sheets.xlsx 'SELECT COUNT(1) AS n FROM {0, "Sheet2"}'"""
want = '[{"n": 700}]'
test("Supports Excel with multiple sheets", to_run, want, sort=True)

# ORC support
to_run = """./dsq ./testdata/orc/test_data.orc 'SELECT COUNT(*) FROM {} WHERE _col8="China"'"""
want = '[{"COUNT(*)":189}]'
test("Supports ORC files", to_run, want, sort=True)

# Avro support
to_run = """./dsq ./testdata/avro/test_data.avro 'SELECT COUNT(*) FROM {} WHERE country="Sweden"'"""
want = '[{"COUNT(*)":25}]'
test("Supports Avro files", to_run, want, sort=True)

# Version test
to_run = """./dsq -v"""
want_stderr = "dsq latest\n"
test("Shows version and quits", to_run, want="", want_stderr=want_stderr)

# Pretty column order
to_run = """./dsq --pretty testdata/path/path.json 'SELECT name, id FROM {"data.data"}'"""
want = """+----+-------+
| id | name  |
+----+-------+
|  1 | Corah |
|  3 | Minh  |
+----+-------+
(2 rows)"""
test("Pretty column order alphabetical", to_run, want)

# Pretty without query
to_run = """./dsq --pretty testdata/regr/36.json"""
want = """+---+---+-------+
| a | b |   c   |
+---+---+-------+
| 1 | 2 | [1,2] |
+---+---+-------+
(1 row)"""
test("Pretty works even without query", to_run, want)

# Prints schema pretty
to_run = """./dsq --pretty --schema testdata/regr/36.json"""
want = """Array of
  Object of
    a of
      number
    b of
      number
    c of
      Array of
        number
"""
test("Pretty prints schema", to_run, want)

# Prints schema as JSON
to_run = """./dsq --schema testdata/regr/36.json"""
want = """{
  "kind": "array",
  "array": {
    "kind": "object",
    "object": {
      "b": {
        "kind": "scalar",
        "scalar": "number"
      },
      "c": {
        "kind": "array",
        "array": {
          "kind": "scalar",
          "scalar": "number"
        }
      },
      "a": {
        "kind": "scalar",
        "scalar": "number"
      }
    }
  }
}"""
test("Prints schema as JSON", to_run, want, sort=True)

# SQL file tests
# Simple sql query from file
to_run = """./dsq  testdata/userdata.json --file ./testdata/sql/simple.sql"""
want = """
[{" Name ":"Michelle Yost"},
{" Name ":"Guadalupe Schimmel II"},
{" Name ":"Corey Beier"}]
"""
test("Run simple query from sql file", to_run, want, sort=True)

# Error when query file is empty
to_run = """./dsq  testdata/userdata.json --file ./testdata/sql/empty.sql"""
want_stderr = "SQL file is empty.\n"
test("Run query from empty sql file", to_run, want="", want_stderr=want_stderr, fail=True)

# Error when query file is empty
to_run = """./dsq  testdata/userdata.json -f"""
want_stderr = "Must specify a SQL file.\n"
test("Not specifying sql file", to_run, want="", want_stderr=want_stderr, fail=True)

# Cache test
# Drop the db file on disk to make sure this test's cache is clean.
for f in glob.glob(os.path.join(tempfile.gettempdir(), "dsq-cache-*.db*")):
    print("Deleting existing dsq database file: " + f)
    os.remove(f)
    
to_run = """
./dsq --cache taxi.csv "SELECT passenger_count, COUNT(*), AVG(total_amount) FROM {} GROUP BY passenger_count ORDER BY COUNT(*) DESC"
"""
want = """
[{"AVG(total_amount)":17.641883306799908,"passenger_count":"1","COUNT(*)":1533197},
{"AVG(total_amount)":18.097587071145647,"passenger_count":"2","COUNT(*)":286461},
{"AVG(total_amount)":32.23715114825533,"passenger_count":"","COUNT(*)":128020},
{"AVG(total_amount)":17.915395871092315,"passenger_count":"3","COUNT(*)":72852},
{"AVG(total_amount)":17.270924817567234,"passenger_count":"5","COUNT(*)":50291},
{"passenger_count":"0","COUNT(*)":42228,"AVG(total_amount)":17.021401676615067},
{"passenger_count":"6","COUNT(*)":32623,"AVG(total_amount)":17.600296416636713},
{"passenger_count":"4","COUNT(*)":25510,"AVG(total_amount)":18.452774990196012},
{"COUNT(*)":2,"AVG(total_amount)":95.705,"passenger_count":"8"},
{"passenger_count":"7","COUNT(*)":2,"AVG(total_amount)":87.17},
{"passenger_count":"9","COUNT(*)":1,"AVG(total_amount)":113.6}]"""
want_stderr = "Cache invalid, re-import required.\n"

test("Caching from file (first time so import is required)", to_run, want, want_stderr=want_stderr, sort=True)

to_run = """
cat taxi.csv | ./dsq --cache -s csv 'SELECT passenger_count, COUNT(*), AVG(total_amount) FROM {} GROUP BY passenger_count ORDER BY COUNT(*) DESC'
"""

test("Caching from pipe (second time so import not required)", to_run, want, sort=True, winSkip=True, within_seconds=5)

to_run = """
cat testdata/taxi_trunc.csv | ./dsq --cache -s csv 'SELECT passenger_count, COUNT(*), AVG(total_amount) FROM {} GROUP BY passenger_count ORDER BY COUNT(*) DESC'"""
want = """[{"COUNT(*)":9,"AVG(total_amount)":20.571111111111115,"passenger_count":"1"},
{"passenger_count":"0","COUNT(*)":1,"AVG(total_amount)":43.67}]"""
want_stderr = "Cache invalid, re-import required.\n"

test("Re-imports when file changes with cache on", to_run, want, want_stderr=want_stderr, sort=True, winSkip=True)

# Mode support
to_run = """./dsq testdata/userdata.json 'SELECT mode(Activated) mostly_activated FROM {}' """
want = '[{"mostly_activated":1}]'
test("Mode support", to_run, want=want)

# URL functions
to_run = """./dsq testdata/basic_logs.csv 'SELECT url_host(request) host, count(1) count FROM {} group by host' """
want = '[{"host":"age.com","count":2}]'
test("URL functions", to_run, want=want, sort=True)

# URL functions, split_part
to_run = """./dsq testdata/basic_logs.csv "SELECT split_part(url_host(request), '.', -1) host, count(1) count FROM {} group by host" """
want = '[{"host":"com","count":2}]'
test("URL functions", to_run, want=want, sort=True)

# Number conversion
to_run = """./dsq testdata/convert.csv 'SELECT * FROM {}'"""
want = """[{"test":"1"},
{"test":"1.1"},
{"test":"+1"},
{"test":"01"},
{"test":"001"},
{"test":"0001.1"}]"""
test("No number conversion, with query", to_run, want=want, sort=True)

to_run = """./dsq --convert-numbers testdata/convert.csv 'SELECT * FROM {}'"""
want = """[{"test":1},
{"test":1.1},
{"test":1},
{"test":1},
{"test":1},
{"test":1.1}]"""
test("Number conversion, with query", to_run, want=want, sort=True)

to_run = """./dsq testdata/convert.csv"""
want = """[{"test":"1"},
{"test":"1.1"},
{"test":"+1"},
{"test":"01"},
{"test":"001"},
{"test":"0001.1"}]"""
test("No number conversion, no query", to_run, want=want, sort=True)

to_run = """./dsq --convert-numbers testdata/convert.csv"""
want = """[{"test":1},
{"test":1.1},
{"test":1},
{"test":1},
{"test":1},
{"test":1.1}]"""
test("Number conversion, no query", to_run, want=want, sort=True)

to_run = """./dsq testdata/csv/numberconvert.csv 'select * from {} where score > "90"'"""
want = """[{"Score": "95", "Name": "Rainer"}]"""
test("No number conversion, does alphabet ordering", to_run, want=want, sort=True)

to_run = """./dsq --convert-numbers testdata/csv/numberconvert.csv 'select * from {} where score > "90"'"""
want = """[{"Name":"Rainer","Score":95},
{"Name":"Fountainer","Score":100}]"""
test("Number conversion, number ordering", to_run, want=want, sort=True)

# END OF TESTS

# START OF REGRESSION TESTS
# Nested array support
to_run = """./dsq ./testdata/regr/36.json 'SELECT c->1 AS secondc FROM {}'"""
want = '[{"secondc": "2"}]'
test("https://github.com/multiprocessio/dsq/issues/36", to_run, want, sort=True)

to_run = """./dsq ./testdata/regr/36.json 'SELECT * FROM {}'"""
want = '[{"a": 1, "b": 2, "c": "[1,2]"}]'
test("https://github.com/multiprocessio/dsq/issues/36", to_run, want, sort=True)

to_run = """./dsq ./testdata/regr/67.jsonl 'SELECT COUNT(1) AS count FROM {}'"""
want = '[{"count": 1}]'
test("https://github.com/multiprocessio/dsq/issues/67", to_run, want, sort=True)

to_run = """./dsq ./testdata/regr/74.csv 'SELECT * FROM {}'"""
want = '[{"a": "1", "a b": "2"}]'
test("https://github.com/multiprocessio/dsq/issues/74", to_run, want, sort=True)


# END OF REGRESSION TESTS

print(f"{tests - failures} of {tests} succeeded.")
if failures > 0:
    sys.exit(1)

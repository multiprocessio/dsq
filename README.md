# Commandline tool for running SQL queries against JSON, CSV, Excel, Parquet, and more.

## Stay in the loop

Since Github doesn't provide a great way for you to learn about new
releases and features, don't just star the repo, join the [mailing
list](https://docs.google.com/forms/d/e/1FAIpQLSfYF3AZivacRrQWanC-skd0iI23ermwPd17T_64Xc4etoL_Tw/viewform).

## About

This is a CLI companion to
[DataStation](https://github.com/multiprocessio/datastation) (a GUI)
for running SQL queries against data files. So if you want the GUI
version of this, check out DataStation.

## Install

Binaries for amd64 (x86_64) are provided for each release.

### macOS Homebrew

`dsq` is available on macOS Homebrew:

```bash
$ brew install dsq
```

### Binaries on macOS, Linux, WSL

On macOS, Linux, and WSL you can run the following:

```bash
$ curl -LO "https://github.com/multiprocessio/dsq/releases/download/0.12.0/dsq-$(uname -s | awk '{ print tolower($0) }')-x64-0.12.0.zip"
$ unzip dsq-*-0.12.0.zip
$ sudo mv dsq /usr/local/bin/dsq
```

Or install manually from the [releases
page](https://github.com/multiprocessio/dsq/releases), unzip and add
`dsq` to your `$PATH`.

### Binaries on Windows (not WSL)

Download the [latest Windows
release](https://github.com/multiprocessio/dsq/releases), unzip it,
and add `dsq` to your `$PATH`.

### Build and install from source

If you are on another platform or architecture or want to grab the
latest release, you can do so with Go 1.18+:

```bash
$ go install github.com/multiprocessio/dsq@latest
```

`dsq` will likely work on other platforms that Go is ported to such as
AARCH64 and OpenBSD, but tests and builds are only run against x86_64
Windows/Linux/macOS.

## Usage

You can either pipe data to `dsq` or you can pass a file name to
it. NOTE: piping data doesn't work on Windows.

If you are passing a file, it must have the usual extension for its
content type.

For example:

```bash
$ dsq testdata.json "SELECT * FROM {} WHERE x > 10"
```

Or:

```bash
$ dsq testdata.ndjson "SELECT name, AVG(time) FROM {} GROUP BY name ORDER BY AVG(time) DESC"
```

### Pretty print

By default `dsq` prints ugly JSON. This is the most efficient mode.

```bash
$ dsq testdata/userdata.parquet 'select count(*) from {}'
[{"count(*)":1000}
]
```

If you want prettier JSON you can pipe `dsq` to `jq`.

```bash
$ dsq testdata/userdata.parquet 'select count(*) from {}' | jq
[
  {
    "count(*)": 1000
  }
]
```

Or you can enable pretty printing with `-p` or `--pretty` in `dsq`
which will display your results in an ASCII table.

```bash
$ dsq --pretty testdata/userdata.parquet 'select count(*) from {}'
+----------+
| count(*) |
+----------+
|     1000 |
+----------+
```

### Piping data to dsq

When piping data to `dsq` you need to set the `-s` flag and specify
the file extension or MIME type.

For example:

```bash
$ cat testdata.csv | dsq -s csv "SELECT * FROM {} LIMIT 1"
```

Or:

```bash
$ cat testdata.parquet | dsq -s parquet "SELECT COUNT(1) FROM {}"
```

### Multiple files and joins

You can pass multiple files to DSQ. As long as they are supported data
files in a valid format, you can run SQL against all files as
tables. Each table can be accessed by the string `{N}` where `N` is the
0-based index of the file in the list of files passed on the
commandline.

For example this joins two datasets of differing origin types (CSV and
JSON).

```bash
$ dsq testdata/join/users.csv testdata/join/ages.json \
  "select {0}.name, {1}.age from {0} join {1} on {0}.id = {1}.id"
[{"age":88,"name":"Ted"},
{"age":56,"name":"Marjory"},
{"age":33,"name":"Micah"}]
```

You can also give file-table-names aliases since `dsq` uses standard
SQL:

```bash
$ dsq testdata/join/users.csv testdata/join/ages.json \
  "select u.name, a.age from {0} u join {1} a on u.id = a.id"
[{"age":88,"name":"Ted"},
{"age":56,"name":"Marjory"},
{"age":33,"name":"Micah"}]
```

### Transforming data to JSON without querying

As a shorthand for `dsq testdata.csv "SELECT * FROM {}"` to convert
supported file types to JSON you can skip the query and the converted
JSON will be dumped to stdout.

For example:

```bash
$ dsq testdata.csv
[{...some csv data...},{...some csv data...},...]
```

### Array of objects nested within an object

DataStation and `dsq`'s SQL integration operates on an array of
objects. If your array of objects happens to be at the top-level, you
don't need to do anything. But if your array data is nested within an
object you can add a "path" parameter to the table reference.

For example if you have this data:

```bash
$ cat api-results.json
{
  "data": {
    "data": [
      {"id": 1, "name": "Corah"},
      {"id": 3, "name": "Minh"}
    ]
  },
  "total": 2
}
```

You need to tell `dsq` that the path to the array data is `"data.data"`:

```bash
$ dsq --pretty api-results.json 'SELECT * FROM {0, "data.data"} ORDER BY id DESC'
+----+-------+
| id | name  |
+----+-------+
|  3 | Minh  |
|  1 | Corah |
+----+-------+
```

You can also use the shorthand `{"path"}` or `{'path'}` if you only have one table:

```bash
$ dsq --pretty api-results.json 'SELECT * FROM {"data.data"} ORDER BY id DESC'
+----+-------+
| id | name  |
+----+-------+
|  3 | Minh  |
|  1 | Corah |
+----+-------+
```

You can use either single or double quotes for the path.

#### Multiple Excel sheets

Excel files with multiple sheets are stored as an object with key
being the sheet name and value being the sheet data as an array of
objects.

If you have an Excel file with two sheets called `Sheet1` and `Sheet2`
you can run `dsq` on the second sheet by specifying the sheet name as
the path:

```bash
$ dsq data.xlsx 'SELECT COUNT(1) FROM {"Sheet2"}'
```

#### Limitation: nested arrays

You cannot specify a path through an array, only objects.

### Nested object values

It's easiest to show an example. Let's say you have the following JSON file called `user_addresses.json`:

```bash
$ cat user_addresses.json
[
  {"name": "Agarrah", "location": {"city": "Toronto", "address": { "number": 1002 }}},
  {"name": "Minoara", "location": {"city": "Mexico City", "address": { "number": 19 }}},
  {"name": "Fontoon", "location": {"city": "New London", "address": { "number": 12 }}}
]
```

You can query the nested fields like so:

```sql
$ dsq user_addresses.json 'SELECT name, "location.city" FROM {}'
```

And if you need to disambiguate the table:

```sql
$ dsq user_addresses.json 'SELECT name, {}."location.city" FROM {}'
```

#### Caveat: PowerShell, CMD.exe

On PowerShell and CMD.exe you must escape inner double quotes with backslashes:

```powershell
> dsq user_addresses.json 'select name, \"location.city\" from {}'
[{"location.city":"Toronto","name":"Agarrah"},
{"location.city":"Mexico City","name":"Minoara"},
{"location.city":"New London","name":"Fontoon"}]
```

#### Nested objects explained

Nested objects are collapsed and their new column name becomes the
JSON path to the value connected by `.`. Actual dots in the path must
be escaped with a backslash. Since `.` is a special character in SQL
you must quote the whole new column name.

#### Limitation: whole object retrieval

You cannot query whole objects, you must ask for a specific path that
results in a scalar value.

For example in the `user_addresses.json` example above you CANNOT do this:

```sql
$ dsq user_addresses.json 'SELECT name, {}."location" FROM {}'
```

Because `location` is not a scalar value. It is an object.

### Nested arrays

Nested arrays are converted to a JSON string when stored in
SQLite. Since SQLite supports querying JSON strings you can access
that data as structured data even though it is a string.

So if you have data like this in `fields.json`:

```json
[
  {"field1": [1]},
  {"field1": [2]},
]
```

You can request the entire field:

```
$ dsq fields.json "SELECT field1 FROM {}" | jq
[
  {
    "field1": "[1]"
  },
  {
    "field1": "[2]",
  }
]
```

#### JSON operators

You can get the first value in the array using SQL JSON operators.

```
$ dsq fields.json "SELECT field1->0 FROM {}" | jq
[
  {
    "field1->0": "1"
  },
  {
    "field1->0": "2"
  }
]
```

### REGEXP

Since DataStation and `dsq` are built on SQLite, you can filter using
`x REGEXP 'y'` where `x` is some column or value and `y` is a REGEXP
string. SQLite doesn't pick a regexp implementation. DataStation and
`dsq` use Go's regexp implementation which is more limited than PCRE2
because Go support for PCRE2 is not yet very mature.

```sql
$ dsq user_addresses.json "SELECT * FROM {} WHERE name REGEXP 'A.*'"
[{"location.address.number":1002,"location.city":"Toronto","name":"Agarrah"}]
```

### Output column order

When emitting JSON (i.e. without the `--pretty` flag) keys within an
object are unordered.

If order is important to you you can filter with `jq`: `dsq x.csv
'SELECT a, b FROM {}' | jq --sort-keys`.

With the `--pretty` flag, column order is purely alphabetical. It is
not possible at the moment for the order to depend on the SQL query
order.

### Dumping inferred schema

For any supported file you can dump the inferred schema rather than
dumping the data or running a SQL query. Set the `--schema` flag to do
this.

The inferred schema is very simple, only JSON types are supported. If
the underlying format (like Parquet) supports finer-grained data types
(like int64) this will not show up in the inferred schema. It will
show up just as `number`.

For example:

```
$ dsq testdata/avro/test_data.avro --schema --pretty
Array of
  Object of
    birthdate of
      string
    cc of
      Varied of
        Object of
          long of
            number or
        Unknown
    comments of
      string
    country of
      string
    email of
      string
    first_name of
      string
    gender of
      string
    id of
      number
    ip_address of
      string
    last_name of
      string
    registration_dttm of
      string
    salary of
      Varied of
        Object of
          double of
            number or
        Unknown
    title of
      string
```

You can print this as a structured JSON string by omitting the
`--pretty` flag when setting the `--schema` flag.

## Supported Data Types

| Name | File Extension(s) | Mime Type | Notes |
|-----------|-|-|--------------------|
| CSV | `csv` | `text/csv` | |
| TSV | `tsv`, `tab` | `text/tab-separated-values` | |
| JSON | `json` | `application/json` | Must be an array of objects or a [path to an array of objects](https://github.com/multiprocessio/dsq#array-of-objects-nested-within-an-object). |
| Newline-delimited JSON | `ndjson`, `jsonl` | `application/jsonlines` ||
| Concatenated JSON | `cjson` | `application/jsonconcat` ||
| ORC | `orc` | `orc` ||
| Parquet | `parquet` | `parquet` ||
| Avro | `avro` || `application/avro` || 
| Excel | `xlsx`, `xls` | `application/vnd.ms-excel` | If you have multiple sheets, you must [specify a sheet path](https://github.com/multiprocessio/dsq#multiple-excel-sheets). |
| ODS | `ods` |`application/vnd.oasis.opendocument.spreadsheet` | If you have multiple sheets, you must [specify a sheet path](https://github.com/multiprocessio/dsq#multiple-excel-sheets). |
| Apache Error Logs | NA | `text/apache2error` | Currently only works if being piped in. |
| Apache Access Logs | NA | `text/apache2access` | Currently only works if being piped in. |
| Nginx Access Logs | NA | `text/nginxaccess` | Currently only works if being piped in. |

## Engine

Under the hood dsq uses
[DataStation](https://github.com/multiprocessio/datastation) as a
library and under that hood DataStation uses SQLite to power these
kinds of SQL queries on arbitrary (structured) data.

## Comparisons

| Name | Link | Supported File Types | Engine |
|----|-|-|------------------------------------------------------------------------|
| q | http://harelba.github.io/q/ | CSV, TSV | SQLite |
| textql | https://github.com/dinedal/textql | CSV, TSV | SQLite |
| octoql | https://github.com/cube2222/octosql | JSON, CSV, Excel, Parquet | Custom engine |
| dsq | Here | CSV, TSV, a few variations of JSON, Parquet, Excel, ODS (OpenOffice Calc), Logs | SQLite |

And many other similar tools:
[sqlite-utils](https://github.com/simonw/sqlite-utils),
[csvq](https://github.com/mithrandie/csvq),
[trdsql](https://github.com/noborus/trdsql).

## Community

[Join us at #dsq on the Multiprocess Discord](https://discord.gg/9BRhAMhDa5).

## How can I help?

Download dsq and use it! Report bugs on
[Discord](https://discord.gg/f2wQBc4bXX).

If you're a developer with some Go experience looking to hack on open
source, check out
[GOOD_FIRST_PROJECTS.md](https://github.com/multiprocessio/datastation/blob/main/GOOD_FIRST_PROJECTS.md)
in the DataStation repo.

## License

This software is licensed under an Apache 2.0 license.

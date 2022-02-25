# Commandline tool for running SQL queries against JSON, CSV, Excel, Parquet, and more.

This is a CLI companion to
[DataStation](https://github.com/multiprocessio/datastation) (a GUI)
for running SQL queries against data files. So if you want the GUI
version of this, check out DataStation.

## Install

Binaries for amd64 (x86_64) are provided for each release.

### macOS, Linux

On macOS or Linux, you can run the following:

```bash
$ curl -LO "https://github.com/multiprocessio/dsq/releases/download/0.4.0/dsq-$(uname -s | awk '{ print tolower($0) }')-x64-0.4.0.zip"
$ unzip -f dsq-*-0.4.0.zip
$ sudo mv dsq /usr/local/bin/dsq
```

Or install manually from the [releases
page](https://github.com/multiprocessio/dsq/releases), unzip and add
`dsq` to your `$PATH`.

### Windows

Download the [latest Windows
release](https://github.com/multiprocessio/dsq/releases), unzip it,
and add `dsq` to your `$PATH`.

### Manual

If you are on another platform or architecture or want to grab the
latest release, you can do so with Go 1.17+:

```
$ go install github.com/multiprocessio/dsq@latest
```

## Usage

You can either pipe data to `dsq` or you can pass a file name to it.

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
$ ./dsq --pretty testdata/userdata.parquet 'select count(*) from {}'
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

### Nested object values

It's easiest to show an example. Let's say you have the following JSON file called `user_addresses.json`:

```json
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

#### Caveat: PowerShell

On PowerShell you must escape inner double quotes with backslashes:

```powershell
> dsq .\testdata\nested\nested.json 'select name, \"location.city\" from {}'
[{"location.city":"Toronto","name":"Agarrah"},
{"location.city":"Mexico City","name":"Minoara"},
{"location.city":"New London","name":"Fontoon"}]
```

#### Nested objects explained

Nested objects are collapsed and their new column name becomes the
JSON path to the value connected by `.`. Actual dots in the path must
be escaped with a backslash. Since `.` is a special character in SQL
you must quote the whole new column name.

#### Limitation: nested arrays

Nested objects within arrays are still ignored/dropped by `dsq`. So if
you have data like this:

```json
[
  {"field1": [1]},
  {"field1": [2]},
]
```

You cannot access any data within `field1`. You will need to
preprocess your data with some other tool.

#### Limitation: whole object retrieval

You cannot query whole objects, you must ask for a specific path that
results in a scalar value.

For example in the `user_addresses.json` example above you CANNOT do this:

```sql
$ dsq user_addresses.json 'SELECT name, {}."location" FROM {}'
```

Because `location` is not a scalar value. It is an object.

## Supported Data Types

| Name | File Extension(s) | Mime Type | Notes |
|-----------|-|-|--------------------|
| CSV | `csv` | `text/csv` | |
| TSV | `tsv`, `tab` | `text/tab-separated-values` | |
| JSON | `json` | `application/json` |  Must be an array of objects. |
| Newline-delimited JSON | `ndjson`, `jsonl` | `application/jsonlines` | |
| Concatenated JSON | `cjson` | `application/jsonconcat` ||
| Parquet | `parquet` | `parquet` ||
| Excel | `xlsx`, `xls` | `application/vnd.ms-excel` | Currently only works if there is only one sheet. |
| ODS | `ods` |`application/vnd.oasis.opendocument.spreadsheet` |  Currently only works if there is only one sheet. |
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

[Join us on Discord](https://discord.multiprocess.io).

## How can I help?

Download the app and use it! Report bugs on
[Discord](https://discord.multiprocess.io).

Before starting on any new feature though, check in on
[Discord](https://discord.multiprocess.io)!

## Subscribe

If you want to hear about new features and how this works under
the hood, [sign up here](https://forms.gle/wH5fdxrxXwZHoNxk8).

## License

This software is licensed under an Apache 2.0 license.

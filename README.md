# Commandline tool for running SQL queries against JSON, CSV, Excel, Parquet, and more.

This is a CLI companion to
[DataStation](https://github.com/multiprocessio/datastation) (a GUI)
for running SQL queries against data files. So if you want the GUI
version of this, check out DataStation.

## Install

Download the [latest
release](https://github.com/multiprocessio/datastation/releases) for
your system, unzip it, and add `dsq` to your $PATH.

Or on macos/Linux:

```bash
$ curl -LO https://github.com/multiprocessio/dsq/releases/download/$version/dsq-$arch-x64-$version.zip
$ unzip ./dsq-*.zip
$ chmod +x ./dsq
$ sudo mv dsq /usr/local/bin/dsq
```

Where `$arch` is `linux` or `darwin` and `$version` is the version you
want to install.

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

## Supported Data Types

| Name | File Extension(s) | Notes |
|-----------|-|---------------------|
| CSV | `csv` ||
| TSV | `tsv`, `tab` ||
| JSON | `json` | Must be an array of objects. Nested object fields are ignored. |
| Newline-delimited JSON | `ndjson`, `jsonl` ||
| Parquet | `parquet` ||
| Excel | `xlsx`, `xls` | Currently only works if there is only one sheet. |
| ODS | `ods` | Currently only works if there is only one sheet. |
| Apache Error Logs | `text/apache2error` | Currently only works if being piped in. |
| Apache Access Logs | `text/apache2access` | Currently only works if being piped in. |
| Nginx Access Logs | `text/nginxaccess` | Currently only works if being piped in. |

## Engine

Under the hood dsq uses
[DataStation](https://github.com/multiprocessio/datastation) as a
library and under that hood DataStation uses SQLite to power these
kinds of SQL queries on arbitrary (structured) data.

## Comparisons

The speed column is based on rough benchmarks based on [q's
benchmarks](https://github.com/harelba/q/blob/master/test/BENCHMARK.md). Eventually
I'll do a more thorough and public benchmark.

| Name | Link | Speed | Supported File Types | Engine |
|----|-|-|-|------------------------------------------------------------------------|
| q | http://harelba.github.io/q/ | Fast | CSV, TSV | Uses SQLite |
| textql | https://github.com/dinedal/textql | Ok | CSV, TSV | Uses SQLite |
| octoql | https://github.com/cube2222/octosql | Slow | JSON, CSV, Excel, Parquet | Custom engine missing many features from SQLite |
| dsq | Here | Ok | CSV, TSV, JSON, Newline-delimited JSON, Parquet, Excel, ODS (OpenOffice Calc), Logs | Uses SQLite |

## Community

[Join us on Discord](https://discord.gg/f2wQBc4bXX).

## How can I help?

Download the app and use it! Report bugs on
[Discord](https://discord.gg/f2wQBc4bXX).

Before starting on any new feature though, check in on
[Discord](https://discord.gg/f2wQBc4bXX)!

## Subscribe

If you want to hear about new features and how DataStation works under
the hood, [sign up here](https://forms.gle/wH5fdxrxXwZHoNxk8).

## License

This software is licensed under an Apache 2.0 license.

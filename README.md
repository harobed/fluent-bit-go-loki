# fluent-bit loki output plugin

[![Build Status](https://travis-ci.org/cosmo0920/fluent-bit-go-loki.svg?branch=master)](https://travis-ci.org/cosmo0920/fluent-bit-go-loki)
[![Build status](https://ci.appveyor.com/api/projects/status/6s9itaxvrkos11sx/branch/master?svg=true)](https://ci.appveyor.com/project/cosmo0920/fluent-bit-go-loki/branch/master)

Windows binaries are available in [release pages](https://github.com/cosmo0920/fluent-bit-go-loki/releases).

This plugin works with fluent-bit's go plugin interface. You can use fluent-bit loki to ship logs into grafana datasource with loki.

The configuration typically looks like:

```graphviz
fluent-bit --> loki --> grafana <-- other grafana sources
```

# Usage

```bash
$ fluent-bit -e /path/to/built/out_loki.so -c fluent-bit.conf
```

# Prerequisites

* Go 1.11+
* gcc (for cgo)

## Building

```bash
$ make
```

### Configuration Options

* `Url`: Url of loki server API endpoint (defalut value: `http://localhost:3100/api/prom/push`)
* `BatchWait`: Waiting time for batch operation (unit: msec) (default value: 10 milliseconds)
* `BatchSize`: Batch size for batch operation (unit: KiB) (default value: 10 KiB)
* `ExtraLabels`: Set of labels to include with every Loki stream  (default: `job="fluent-bit"`)
* `LabelKeys`: Comma separated list of keys to use as stream labels. <br />
               All other keys will be placed into the log line.
* `LineFormat`: Format to use when flattening the record to a log line. Valid values are "json" or "key_value". If set to "json" the log line sent to Loki will be the fluentd record (excluding any keys extracted out as labels) dumped as json. If set to "key_value", the log line will be each item in the record concatenated together (separated by a single space) in the format <key>=<value>.
* `DropSingleKey`: if set to true and after extracting label_keys a record only has a single key remaining, the log line sent to Loki will just be the value of the record key.

Example:

add this section to fluent-bit.conf

```properties
[Output]
    Name loki
    Match *
    Url http://localhost:3100/api/prom/push
    BatchWait 10 # (10msec)
    BatchSize 30 # (30KiB)
    # interpreted as {test="fluent-bit-go", lang="Golang"}
    Labels {"labels": [{"key": "test", "label": "fluent-bit-go"},{"key": "lang", "label": "Golang"}]}
```

## Useful links

* [fluent-bit-go](https://github.com/fluent/fluent-bit-go)

# Logrus custom text formatter
[![Build Status](https://travis-ci.org/albenik/logrus-text-formatter.svg?branch=master)](https://travis-ci.org/albenik/logrus-text-formatter)

[Logrus](https://github.com/sirupsen/logrus) formatter inspired by [github.com/x-cray/logrus-prefixed-formatter](https://github.com/x-cray/logrus-prefixed-formatter). 

**WARING!!!**

This formatter does not following not original nor x-cray's log format conventions.

Default behaviour of this formatter is adapted to my own need.

## Installation
To install formatter, use `go get`:

```sh
$ go get github.com/albenik/logrus-text-formatter
```

## Usage examples

```go
package main

import (
	"github.com/sirupsen/logrus"
	"github.com/albenik/logrus-text-formatter"
)

var log = logrus.New()

func init() {
	log.Formatter = new(textformatter.Instance)
	log.Level = logrus.DebugLevel
}

func main() {
	log.WithFields(logrus.Fields{
		"__p":    "textformatter",
		"__f":    "main",
		"__t":    "unique-batch-identifier",
		"animal": "walrus",
		"number": 8,
	}).Debug("Started observing beach")

	log.WithFields(logrus.Fields{
		"__p":         "sensor",
		"__t":         "onchange",
		"temperature": -4,
	}).Info("Temperature changes")
}
```

# License
MIT

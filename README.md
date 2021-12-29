# liblog

Yet another simple logging library for Golang.

## Usage

```go
package main

import (
	log "github.com/wimark/liblog"
)

func main() {
	var appName = "super-app"
	log.InitSingleStr(appName) // singletone init
	
	log.Debug("debug msg") // LOGLEVEL = 0 will show this message
	log.Info("info msg") // basic level is Info
	log.Warning("warning msg: %s", "some-text") // full fmt.Printf style
	log.Error("simple error: %s", "some-error-text")  // logging is channel-based with async call  for error need to Wait / Sleep
}
```

## Copyright

Wimark Systems, 2021
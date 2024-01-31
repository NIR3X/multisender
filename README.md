# MultiSender - Go Module for Simultaneous Writing to Multiple io.Writer Instances

**MultiSender** is a Go module that enables simultaneous writing to multiple `io.Writer` instances. It provides a `MultiSenderWriter` type and a `MultiSender` type to facilitate efficient and parallelized writing.

## Installation

To use this module in your Go project, you can run:

```bash
go get github.com/NIR3X/multisender
```

## Usage

```go
package main

import (
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/NIR3X/filecache"
	"github.com/NIR3X/filewatcher"
	"github.com/NIR3X/logger"
)

const (
	fileCacheMaxSize          = 2 * 1024 * 1024
	fileWatcherUpdateInterval = 1 * time.Second
	rootDir                   = "www"
)

func main() {
	fileCache := filecache.NewFileCache(fileCacheMaxSize)
	multiSender := NewMultiSender(fileCache)
	fileWatcher := filewatcher.NewFileWatcher(fileWatcherUpdateInterval, func(path string, isDir bool) { // created
		if isDir {
			return
		}
		err := fileCache.Update(path)
		if err != nil {
			logger.Eprintln(err)
		}
	}, func(path string, isDir bool) { // removed
		if isDir {
			return
		}
		fileCache.Delete(path)
	}, func(path string, isDir bool) { // modified
		if isDir {
			return
		}
		err := fileCache.Update(path)
		if err != nil {
			logger.Eprintln(err)
		}
	})
	defer fileWatcher.Close()
	err := fileWatcher.Watch(rootDir)
	if err != nil {
		logger.Eprintln(err)
		return
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}
		path = filepath.Join(rootDir, path)
		reader, ident := fileCache.GetCached(path)
		switch ident {
		case filecache.Cached:
			_, err = io.Copy(w, reader)
			if err != nil {
				logger.Eprintln(err)
			}
		case filecache.Piped:
			multiSenderWriter := multiSender.Add(path, w)
			multiSenderWriter.Wait()
		}
	})
	logger.Println("Listening on port 8000...")
	err = http.ListenAndServe(":8000", nil)
	if err != nil {
		logger.Eprintln(err)
		return
	}
}
```

## License
[![GNU AGPLv3 Image](https://www.gnu.org/graphics/agplv3-155x51.png)](https://www.gnu.org/licenses/agpl-3.0.html)  

This program is Free Software: You can use, study share and improve it at your
will. Specifically you can redistribute and/or modify it under the terms of the
[GNU Affero General Public License](https://www.gnu.org/licenses/agpl-3.0.html) as
published by the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

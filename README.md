# Soforo

Soforo is a Go package that helps with the implementation of a pluggable Driver
and Repository model for your application. It is inspired by how the 
`database/sql` package works in Go.

In the `database/sql` package, there are generic interfaces that represent a
connection to a database. You can then use these interfaces to perform
operations on the database. The `database/sql` package also provides a way to
register drivers for different databases. This way, you can use the same
interface to interact with different databases.

In the same way, Soforo provides a way to define type of connections and a
repository interface that can be implemented by different drivers. You can then
use the repository interface to interact with the underlying connection.

Soforo does not manage the connections for you. It is up to the driver to
implement the connection management. Soforo only provides a way to define the
interface for the connection and the repository.

Soforo assumes that the connection specification is a URL. The scheme of the URL
is used to determine the driver to use. The rest of the URL is passed to the
driver to establish the connection.

## Usage

To use Soforo, you need to define a connection type and a repository interface.
You can then implement the connection type and the repository interface for
different drivers.

Here is an example of how you can use Soforo to interact with a file system.

```go

package storage

import (
	"github.com/amicolabs/soforo"
	"io/fs"
)

type Driver interface {
	soforo.Driver[Repository]
}

type Repository interface {
	soforo.Repository
	fs.FS
	fs.ReadFileFS

	WriteFile(name string, data []byte) error
}

var Drivers = soforo.NewDrivers[Driver, Repository]("storage")

// This interface should be implemented by the object that is passed to the 
// Open method of the driver of connection types that need a storage repository.
type Provider interface {
	Storage() Repository
}
```

In the example above, we define a `Driver` interface that extends the
`driver.Driver` interface and a `Repository` interface that extends the
`soforo.Repository`, `fs.FS` and `fs.ReadFileFS` interfaces. We then create a
new driver registry using `soforo.NewDrivers`.

Next, we can implement the driver and the repository for the file system.

```go

package file

import (
	"fmt"
    "io/fs"
    "net/url"
    "os"
    "path"

	"github.com/amicolabs/soforo/examples/storage"
)

type Driver struct{}

func (d Driver) Open(url *url.URL, _ interface{}) (storage.Repository, error) {
	p := url.Path
	if p == "" {
		p = url.Opaque
	}
	fsys, ok := os.DirFS(p).(FS)

	if !ok {
		return nil, fmt.Errorf("opening file system %s failed", url.Path)
	}

	return &Repository{path: p, fs: fsys}, nil
}

func (d Driver) Dependencies() []string {
	return []string{}
}

type FS interface {
	fs.FS
	fs.StatFS
	fs.ReadFileFS
	fs.ReadDirFS
}

type Repository struct {
	path string
	fs   FS
}

func (r *Repository) Open(name string) (fs.File, error) {
	return r.fs.Open(name)
}

func (r *Repository) Stat(name string) (fs.FileInfo, error) {
	return r.fs.Stat(name)
}

func (r *Repository) ReadFile(name string) ([]byte, error) {
	return r.fs.ReadFile(name)
}

func (r *Repository) ReadDir(name string) ([]fs.DirEntry, error) {
	return r.fs.ReadDir(name)
}

func (r *Repository) WriteFile(name string, data []byte) error {
	return os.WriteFile(path.Join(r.path, name), data, 0644)
}

func (r *Repository) Close() error {
	return nil
}

func init() {
	storage.Drivers.Register("file", Driver{})
}
```

This is an example of a file system driver. The driver implements the `Driver`
interface and the repository implements the `Repository` interface.

This is how you can use the file system driver.

```go

package main

import (
    "fmt"
    "net/url"

    "github.com/amicolabs/soforo/examples/storage"
    _ "github.com/amicolabs/soforo/examples/storage/file"
)

func main() {
    u, _ := url.Parse("file:///tmp")
    repo, _ := storage.Drivers.Open(u)
    data, _ := repo.ReadFile("test.txt")
    fmt.Println(string(data))
}
```

This exact interface could also be implemented for S3, etc. Using this model
the underlying storage can be swapped out without changing the code that uses
the repository interface.
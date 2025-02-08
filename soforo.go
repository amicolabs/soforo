// © 2025 Rolf van de Krol <rolf@vandekrol.xyz>

// Package soforo provides a generic interface around drivers for various
// connection types. It can be used for databases, storage and other connection
// types.
package soforo

import (
	"fmt"
	"log"
	"net/url"
	"sort"
	"sync"
)

// Repository the value that eventually ends up in the application. This
// interface should be extended by the consuming package to include the methods
// that the repository provides.
type Repository interface {
	Close() error
}

// Driver is the interface that must be implemented by a driver implementation.
// This interface is generic and should be extended by the consuming package to
// specify the type of Repository that the driver provides. In most cases, the
// Driver interface in the consuming packages won't need extra methods.
type Driver[R Repository] interface {
	// Open returns a new Repository or an error. It is the caller's
	// responsibility to call Close on Repository when the Repository is no
	// longer needed. The provider argument is used to provide the driver with
	// additional context. Some drivers need other services to be available in
	// the provider struct. The driver must verify that the provided
	// instance is of the expected type and return an error if it is not.
	Open(url *url.URL, provider interface{}) (R, error)
}

// Drivers is a collection of drivers that can be used to open a repository.
// The drivers are stored in a map and can be registered using the Register
// method. The Open method can be used to open a repository using the URL.
// The URL scheme is used to determine which driver to use.
// The Drivers collection is safe for concurrent use, because it uses a mutex
// to synchronize access to the map.
type Drivers[D Driver[R], R Repository] struct {
	name    string
	drivers map[string]D
	mu      sync.RWMutex
}

// NewDrivers returns a new Drivers collection.
func NewDrivers[D Driver[R], R Repository](name string) *Drivers[D, R] {
	return &Drivers[D, R]{
		name:    name,
		drivers: make(map[string]D),
	}
}

// Register makes a driver available by the provided name.
// If Register is called twice with the same name it panics.
func (ds *Drivers[D, R]) Register(name string, driver D) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if _, dup := ds.drivers[name]; dup {
		log.Panicf("Register called twice for %s driver %s", ds.name, name)
	}
	ds.drivers[name] = driver
}

// Drivers returns a sorted list of the names of the registered drivers.
func (ds *Drivers[D, R]) Drivers() []string {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	list := make([]string, 0, len(ds.drivers))
	for name := range ds.drivers {
		list = append(list, name)
	}
	sort.Strings(list)

	return list
}

// Driver returns the driver with the provided name. If the driver is not
// registered, an error is returned. It uses the URL scheme to determine which
// driver to use.
func (ds *Drivers[D, R]) Driver(u *url.URL) (D, error) {
	if !u.IsAbs() {
		var d D
		return d, fmt.Errorf("invalid database source name %q", u.String())
	}

	return ds.DriverByName(u.Scheme)
}

// DriverByName returns the driver with the provided name. If the driver is not
// registered, an error is returned.
func (ds *Drivers[D, R]) DriverByName(name string) (D, error) {
	ds.mu.RLock()
	driver, ok := ds.drivers[name]
	ds.mu.RUnlock()

	if !ok {
		var d D
		return d, fmt.Errorf("unknown driver %q (forgotten import?)", name)
	}
	return driver, nil
}

// Open opens a Repository from a driver using the URL. It is the caller's
// responsibility to call Close on Repository when the Repository is no longer
// needed. Open uses the URL scheme to determine which driver to use. If the
// driver is unknown, Open returns an error. The provider argument is used to
// provide the driver with additional context. The driver must verify that
// the provided instance is of the expected type and return an error if it is
// not.
func (ds *Drivers[D, R]) Open(u *url.URL, provider interface{}) (R, error) {
	driver, err := ds.Driver(u)
	if err != nil {
		var r R
		return r, err
	}

	return driver.Open(u, provider)
}

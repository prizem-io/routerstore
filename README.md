# routerstore [![Build Status](https://travis-ci.org/prizem-io/routerstore.svg?branch=master)](https://travis-ci.org/prizem-io/routerstore) [![Coverage Status](https://coveralls.io/repos/github/prizem-io/routerstore/badge.svg?branch=master)](https://coveralls.io/github/prizem-io/routerstore?branch=master) [![GoDoc](https://godoc.org/github.com/prizem-io/routerstore?status.svg)](http://godoc.org/github.com/prizem-io/routerstore) [![Go Report](https://goreportcard.com/badge/github.com/prizem-io/routerstore)](https://goreportcard.com/report/github.com/prizem-io/routerstore)

RouterStore is a lightweight high performance HTTP request router and data store for [Go](https://golang.org/).  It registers requests patterns with any kind of data and retrieves the data with matching request paths.

Typical routers (often called *multiplexers* or *muxes*) are focused on providing a high performance alternative to [Go's stardard HTTP library](https://golang.org/pkg/net/http/#ServeMux) in the `net/http` package.  If that is your need, there are several [great libraries to choose from](https://github.com/julienschmidt/go-http-routing-benchmark).  This library is specifically used for resolving application data for matching HTTP request paths.  For example, the data plane of a [service mesh](https://buoyant.io/2017/04/25/whats-a-service-mesh-and-why-do-i-need-one/) would need to find the service information for each incoming request in order to proxy the network traffic.

This router's algorithm uses the similar principles as others to achieve high performance, small memory footprint, and scalability.  A radix tree structure and zero memory allocation implementation provides performance comparable to the very fast [httprouter library](https://github.com/julienschmidt/httprouter).  In addition, this router implements desirable features for service meshes and API gateways.

## Features

**Prioritized matching:** Other routers can return unexpected results when a request path can match multiple routes.  For example, the router might return the first matching route when a more specific different route is a better match.  This router determines the best match by applying the following priority order:

1. Static routes
2. Regular expression variable routes (in order of registration)
3. Simple variable routes
4. Wildcard routes (when no child routes match)

**Non-conflicting path variables:** Sometimes, an API might need to name a path variable differently.  E.g.:

* `GET /people/:id` => Return the person by their identifier
* `GET /people/:id/tasks` => Returns the person's tasks
* `GET /people/:last/:first` => Find the person by their first and last names

The router's match result provides the correct variable name and values for the provided request path.  E.g.:

```
GET /people/1234/tasks
Returns id = 1234
```

```
GET /people/doe/john
Returns first = john, last = doe
```

**Great performance:** This is attributed to the radix tree structure for building routes and that the matching logic allocates 0 bytes of heap.  Not even path parameters create garbage.

**Bad routes return errors:** This is a small but important detail.  Other routers will panic when the developer tried to add a bad route.  This makes sense for an API, but not for a proxy that needs to handle bad configuration.

## Usage

```
package main

import (
	"log"

	"github.com/prizem-io/routerstore"
)

type Service struct {
	Name    string
	Version string
}

func main() {
	helloService := Service{
		Name:    "Hello",
		Version: "V1",
	}
	router := routerstore.New()
	router.GET("/people", &helloService)
	router.GET("/people/:id", &helloService)
	router.GET("/people/:id/tasks", &helloService)
	router.GET("/people/:last/:first", &helloService)

	var result routerstore.Result
	var err error
	var service *Service

	log.Printf("GET /people/1234")
	err = router.Match(routerstore.GET, "/people/1234", &result)
	service = checkResult(&result, err)
	log.Printf("\tService %s, Version %s", service.Name, service.Version)
	log.Printf("\tPerson ID: %s", result.Param("id"))

	log.Printf("GET /people/doe/john")
	err = router.Match(routerstore.GET, "/people/doe/john", &result)
	service = checkResult(&result, err)
	log.Printf("\tService %s, Version %s", service.Name, service.Version)
	log.Printf("\tPerson Name: %s %s", result.Param("first"), result.Param("last"))
}

func checkResult(result *routerstore.Result, err error) *Service {
	if err != nil {
		log.Fatalf("could not match: %v", err)
	}
	service, ok := result.Data.(*Service)
	if !ok {
		log.Fatal("Unexpected result data")
	}
	return service
}
```
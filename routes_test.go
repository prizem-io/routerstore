// Copyright 2017 The Prizem Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package routerstore

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	resources   []string
	collections []string
	searches    []string
	entities    []string
	result      Result
)

func TestMain(m *testing.M) {
	resources = []string{"customers", "products", "carts", "wines", "bottles", "cellars", "locations", "widgets", "people", "places", "things", "foo", "bar", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine", "ten", "eleven", "twelve", "thirteen", "fourteen", "fifteen", "sixteen", "seventeen", "eighteen", "nineteen", "twenty", "twenty-one", "twenty-two", "twenty-three", "twenty-four", "twenty-five"}
	collections = make([]string, len(resources))
	searches = make([]string, len(resources))
	entities = make([]string, len(resources))

	for i, resource := range resources {
		collections[i] = fmt.Sprintf("/%s", resource)
		searches[i] = fmt.Sprintf("/%s/search", resource)
		entities[i] = fmt.Sprintf("/%s/:id", resource)
	}

	os.Exit(m.Run())
}

func TestRouteOk(t *testing.T) {
	details := "details"
	var handler RouteMux

	handler.GET("/person/:last/:first/", details)
	registerResources(&handler, resources)

	err := handler.Match("GET", "/person/anderson/thomas/", &result)
	require.Nil(t, err)
	assert.Equal(t, details, result.Data)
	assert.Equal(t, "anderson", result.Param("last"))
	assert.Equal(t, "thomas", result.Param("first"))
	assert.Equal(t, "", result.Param("unknown"))

	testResources(t, &handler, resources)
}

func TestDuelingVariables(t *testing.T) {
	details1 := "details1"
	details2 := "details2"
	details3 := "details3"
	details4 := "details4"
	var handler RouteMux

	handler.GET("/people/:number([0-9]+)/test", details1)
	handler.GET("/people/:number([0-9]+)/:other", details2)
	handler.GET("/people/:last/:first/", details3)
	handler.GET("/people/:id/test", details4)

	err := handler.Match("GET", "/people/abcd/test", &result)
	require.Nil(t, err)
	assert.Equal(t, details4, result.Data)
	assert.Equal(t, "abcd", result.Param("id"))
	assert.Equal(t, "", result.Param("unknown"))

	err = handler.Match("GET", "/people/1234/test", &result)
	require.Nil(t, err)
	assert.Equal(t, details1, result.Data)
	assert.Equal(t, "1234", result.Param("number"))
	assert.Equal(t, "", result.Param("other"))
	assert.Equal(t, "", result.Param("unknown"))

	err = handler.Match("GET", "/people/1234/5678", &result)
	require.Nil(t, err)
	assert.Equal(t, details2, result.Data)
	assert.Equal(t, "1234", result.Param("number"))
	assert.Equal(t, "5678", result.Param("other"))
	assert.Equal(t, "", result.Param("unknown"))

	err = handler.Match("GET", "/people/anderson/thomas", &result)
	require.Nil(t, err)
	assert.Equal(t, details3, result.Data)
	assert.Equal(t, "anderson", result.Param("last"))
	assert.Equal(t, "thomas", result.Param("first"))
	assert.Equal(t, "", result.Param("unknown"))
}

func TestWildcard(t *testing.T) {
	details := "details"
	var handler RouteMux

	handler.GET("/person/:id([0-9]+)/contacts", details)
	err := handler.GET("/person/*/test", details)
	assert.Equal(t, ErrWildcardMisplaced, err)

	err = handler.GET("/person/*", details)
	assert.Nil(t, err)

	err = handler.Match("GET", "/person/anderson", &result)
	require.Nil(t, err)
	assert.Equal(t, details, result.Data)
	assert.Equal(t, "anderson", result.Param("*"))
	assert.Equal(t, "", result.Param("unknown"))

	err = handler.Match("GET", "/person/anderson/thomas", &result)
	require.Nil(t, err)
	assert.Equal(t, details, result.Data)
	assert.Equal(t, "anderson/thomas", result.Param("*"))
	assert.Equal(t, "", result.Param("unknown"))
}

func TestBadRegexp(t *testing.T) {
	details := "details"
	handler := New()

	err := handler.GET("/customers/:id(.", details)
	assert.True(t, strings.HasPrefix(err.Error(), "error parsing regexp"))
}

func TestBadSyntax(t *testing.T) {
	details := "details"
	handler := New()

	err := handler.GET("/customers//:test", details)
	assert.Equal(t, ErrBadSyntax, err)
}

func TestNotFound(t *testing.T) {
	details := "details"
	handler := New()

	handler.GET("/person/:last([a-z]+)/:first", details)
	err := handler.Match("POST", "/", &result)
	assert.Equal(t, ErrNotFound, err)

	err = handler.Match("GET", "/person/test", &result)
	assert.Equal(t, ErrNotFound, err)

	err = handler.Match("GET", "/person/1234/test", &result)
	assert.Equal(t, ErrNotFound, err)
}

func Benchmark_Details_Collection(b *testing.B) {
	handler := New()
	registerResources(handler, resources)
	l := len(collections)

	for i := 0; i < b.N; i++ {
		r := i % l
		handler.Match("GET", collections[r], &result)
	}
}

func Benchmark_Details_Search(b *testing.B) {
	handler := New()
	registerResources(handler, resources)
	l := len(searches)

	for i := 0; i < b.N; i++ {
		r := i % l
		handler.Match("GET", searches[r], &result)
	}
}

func Benchmark_Details_Entity(b *testing.B) {
	handler := New()
	registerResources(handler, resources)
	l := len(entities)

	for i := 0; i < b.N; i++ {
		r := i % l
		handler.Match("GET", entities[r], &result)
	}
}

func registerResources(handler *RouteMux, resources []string) {
	for _, resource := range resources {
		collectionPath := fmt.Sprintf("/%s", resource)
		entityPath := fmt.Sprintf("/%s/:id", resource)
		post := fmt.Sprintf("POST %s", resource)
		handler.POST(collectionPath, post)
		getAll := fmt.Sprintf("GET %s", resource)
		handler.GET(collectionPath, getAll)
		search := fmt.Sprintf("GET %s/search", resource)
		handler.GET(fmt.Sprintf("/%s/search", resource), search)
		get := fmt.Sprintf("GET %s/:id", resource)
		handler.GET(entityPath, get)
		put := fmt.Sprintf("PUT %s/:id", resource)
		handler.PUT(entityPath, put)
		patch := fmt.Sprintf("PATCH %s/:id", resource)
		handler.PATCH(entityPath, patch)
		delete := fmt.Sprintf("DELETE %s/:id", resource)
		handler.DELETE(entityPath, delete)
	}
}

func testResources(t *testing.T, handler *RouteMux, resources []string) {
	for _, resource := range resources {
		collectionPath := fmt.Sprintf("/%s", resource)
		entityPath := fmt.Sprintf("/%s/1", resource)

		post := fmt.Sprintf("POST %s", resource)
		err := handler.Match(POST, collectionPath, &result)
		require.Nil(t, err)
		require.NotNil(t, result.Data)
		assert.Equal(t, post, result.Data)

		getAll := fmt.Sprintf("GET %s", resource)
		err = handler.Match(GET, collectionPath, &result)
		require.Nil(t, err)
		require.NotNil(t, result.Data)
		assert.Equal(t, getAll, result.Data)

		search := fmt.Sprintf("GET %s/search", resource)
		err = handler.Match(GET, fmt.Sprintf("/%s/search", resource), &result)
		require.Nil(t, err)
		assert.Equal(t, search, result.Data)

		get := fmt.Sprintf("GET %s/:id", resource)
		err = handler.Match(GET, entityPath, &result)
		require.Nil(t, err)
		require.NotNil(t, result.Data)
		assert.Equal(t, get, result.Data)

		put := fmt.Sprintf("PUT %s/:id", resource)
		err = handler.Match(PUT, entityPath, &result)
		require.Nil(t, err)
		require.NotNil(t, result.Data)
		assert.Equal(t, put, result.Data)

		patch := fmt.Sprintf("PATCH %s/:id", resource)
		err = handler.Match(PATCH, entityPath, &result)
		require.Nil(t, err)
		require.NotNil(t, result.Data)
		assert.Equal(t, patch, result.Data)

		delete := fmt.Sprintf("DELETE %s/:id", resource)
		err = handler.Match(DELETE, entityPath, &result)
		require.Nil(t, err)
		require.NotNil(t, result.Data)
		assert.Equal(t, delete, result.Data)
	}
}

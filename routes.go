// Copyright 2017 The Prizem Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package routerstore

import (
	"errors"
	"regexp"
	"strings"
)

// HTTP method string values
const (
	CONNECT = "CONNECT"
	DELETE  = "DELETE"
	GET     = "GET"
	HEAD    = "HEAD"
	OPTIONS = "OPTIONS"
	PATCH   = "PATCH"
	POST    = "POST"
	PUT     = "PUT"
	TRACE   = "TRACE"
)

var (
	// ErrNotFound denotes that a route for a given request path was not found.
	ErrNotFound = errors.New("Route not found")
	// ErrBadSyntax denotes that an invalid pattern was passed.
	ErrBadSyntax = errors.New("Path contained invalid syntax")
	// ErrWildcardMisplaced denotes that a wildcard was encountered before the end of the pattern.
	ErrWildcardMisplaced = errors.New("Wildcard must be at the end of the path")
)

type (
	route struct {
		// Static routes
		indices   []string
		static    []*route
		staticMap map[string]*route

		// Regex/Variable routes
		variables []*variableRoute
		variable  *route

		// Result fields
		data       interface{}
		paramNames []string

		// Wildcard flag
		wildcard bool
	}

	variableRoute struct {
		expr  string
		route *route
		regex *regexp.Regexp
	}

	// RouteMux stores root level routes per HTTP method.
	RouteMux struct {
		methods map[string]*route
	}

	// Param encapsulates a name/value pair.
	Param struct {
		Name  string
		Value string
	}

	// Result encapsulates the details and path parameters returned by the Details method.
	Result struct {
		Data   interface{}
		Params []Param
		params [10]Param // internal array that initially backs Params to prevent allocations
	}
)

// New creates a new RouteMux
func New() *RouteMux {
	return &RouteMux{
		methods: make(map[string]*route, 10),
	}
}

// GET adds a new route for GET requests.
func (m *RouteMux) GET(pattern string, details interface{}) error {
	return m.AddRoute(GET, pattern, details)
}

// PUT adds a new route for PUT requests.
func (m *RouteMux) PUT(pattern string, details interface{}) error {
	return m.AddRoute(PUT, pattern, details)
}

// DELETE adds a new route for DELETE requests.
func (m *RouteMux) DELETE(pattern string, details interface{}) error {
	return m.AddRoute(DELETE, pattern, details)
}

// PATCH adds a new route for PATCH requests.
func (m *RouteMux) PATCH(pattern string, details interface{}) error {
	return m.AddRoute(PATCH, pattern, details)
}

// POST adds a new route for POST requests.
func (m *RouteMux) POST(pattern string, details interface{}) error {
	return m.AddRoute(POST, pattern, details)
}

// AddRoute adds a new route to that stores to the provided data.
func (m *RouteMux) AddRoute(method string, pattern string, data interface{}) error {
	// Remove leading and trailing slashes and split the url into sections.
	l := len(pattern)
	for l > 0 && pattern[0] == '/' {
		pattern = pattern[1:]
		l--
	}
	for l > 0 && pattern[l-1] == '/' {
		pattern = pattern[:l-1]
		l--
	}

	// Initialize methods map, if needed.
	if m.methods == nil {
		m.methods = make(map[string]*route, 10)
	}

	// Get root route from method map.
	r, ok := m.methods[method]
	if !ok {
		r = &route{}
		m.methods[method] = r
	}

	if l == 0 {
		r.data = data
		return nil
	}

	parts := strings.Split(pattern, "/")

	// Check for misplaced wildcard parts.
	for i, part := range parts {
		if part == "*" && i != len(parts)-1 {
			return ErrWildcardMisplaced
		}
	}

	// Create a slice to capture path parameter names.
	var paramNames = make([]string, 0, 10)

walk:
	for _, part := range parts {
		if len(part) == 0 {
			return ErrBadSyntax
		}

		// Find params that start with ":" and create variable routes.
		if part[0] == ':' {
			// Variable part
			expr := ""
			// A user may choose to override the defult expression
			// similar to expressjs: ‘/user/:id([0-9]+)’
			if index := strings.Index(part, "("); index != -1 {
				expr = part[index:]
				part = part[:index]
			}

			paramNames = append(paramNames, part[1:])

			if expr == "" {
				// No custom regexp defined.
				if r.variable != nil {
					r = r.variable
					continue walk
				}

				// Set the non-regexp variable route.
				next := &route{}
				r.variable = next
				r = next
			} else {
				// Find existing regexp.
				for _, v := range r.variables {
					if v.expr == expr {
						r = v.route
						continue walk
					}
				}

				// Compile the new expression.
				regex, regexErr := regexp.Compile(expr)
				if regexErr != nil {
					return regexErr
				}

				// Create the new variable route.
				next := &variableRoute{
					expr:  expr,
					route: &route{},
					regex: regex,
				}

				// Initialize the variable routes slice, if needed.
				if r.variables == nil {
					r.variables = make([]*variableRoute, 0, 1)
				}

				r.variables = append(r.variables, next)
				r = next.route
			}
		} else if part == "*" {
			// Wildcard part
			r.wildcard = true
		} else {
			// Static part
			var next *route

			// If non-nil, use the static map to find an existing route.
			if r.staticMap != nil {
				next = r.staticMap[part]
				if next == nil {
					next = &route{}
				}

				r.staticMap[part] = next
			} else {
				// Initialize the parallel slices, if needed.
				if r.indices == nil {
					r.indices = make([]string, 0, 10)
					r.static = make([]*route, 0, 10)
				}

				// Find existing static route.
				for i, v := range r.indices {
					if v == part {
						next = r.static[i]
						break
					}
				}

				// A new route must be created.
				if next == nil {
					next = &route{}

					// Is the count of static routes enough to warrant a map instead of a slice.
					if len(r.indices) >= 5 {
						// Convert the parallel slices to a map.
						r.staticMap = make(map[string]*route, 25)
						for i, v := range r.indices {
							r.staticMap[v] = r.static[i]
						}

						r.staticMap[part] = next

						// Allow slices to be GC'd
						r.indices = nil
						r.static = nil
					} else {
						r.indices = append(r.indices, part)
						r.static = append(r.static, next)
					}
				}
			}

			r = next
		}
	}

	// Set the data and parameter names.
	r.data = data
	r.paramNames = paramNames

	return nil
}

// Match tries to find a match for the provided method and request path.   If a match is found,
// details and path paramater values are set in result.
func (m *RouteMux) Match(method string, requestPath string, result *Result) error {
	result.Data = nil
	result.Params = result.params[:0]

	// Get the root route from the methods map.
	r, ok := m.methods[method]
	if !ok {
		return ErrNotFound
	}

	// Remove leading and trailing slashes from the request path.
	l := len(requestPath)
	for l > 0 && requestPath[0] == '/' {
		requestPath = requestPath[1:]
		l--
	}
	for l > 0 && requestPath[l-1] == '/' {
		requestPath = requestPath[:l-1]
		l--
	}

	if l == 0 {
		result.Data = r.data
		return nil
	}

	wildcard := false
	var wildcardResult Result
	var wildcardPath string

pathloop:
	for {
		// If this is a wildcard route, make copies of the state in case child routes don't match.
		if r.wildcard {
			wildcard = true
			wildcardResult.Params = result.Params
			wildcardResult.Data = r.data
			wildcardPath = requestPath
		}

		// Extract next path part.
		part := requestPath
		index := strings.IndexByte(requestPath, '/')
		if index > 0 {
			part = part[0:index]
			requestPath = requestPath[index+1:]
		}

		// Test static routes slice for a match.
		for i, value := range r.indices {
			if value == part {
				r = r.static[i]

				if index < 0 {
					result.Data = r.data
					break pathloop
				}

				continue pathloop
			}
		}

		// Test static routes map for a match.
		if r.staticMap != nil {
			sr, ok := r.staticMap[part]
			if ok {
				r = sr

				if index < 0 {
					result.Data = r.data
					break pathloop
				}

				continue
			}
		}

		// Default to variable route but test regexp routes.
		next := r.variable
		for _, varRoute := range r.variables {
			if varRoute.regex.MatchString(part) {
				next = varRoute.route
				break
			}
		}

		// Not found, break out.
		if next == nil {
			break pathloop
		}

		// Append the path parameter value.
		result.Params = append(result.Params, Param{
			// Name is set below when the route is identified
			Value: part,
		})
		r = next

		// Break out if this is the last path part.
		if index < 0 {
			result.Data = r.data
			break pathloop
		}
	}

	// Revert result to wildcard if the path did not match a more specific route.
	if result.Data == nil && wildcard {
		*result = wildcardResult
		result.Params = append(result.Params, Param{
			Name:  "*",
			Value: wildcardPath,
		})
	}

	// If a match was found, set all the path parameter names and return successfully.
	if result.Data != nil {
		for i, name := range r.paramNames {
			result.Params[i].Name = name
		}
		return nil
	}

	// No match was found.
	return ErrNotFound
}

// Param returns the path parameter value for a given name.
func (r *Result) Param(name string) string {
	for _, p := range r.Params {
		if p.Name == name {
			return p.Value
		}
	}
	return ""
}

package stdlib

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

var ServerBuiltins = map[string]*object.Builtin{
	// router creates a new empty Router struct.
	"server.router": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes no arguments", errors.Ident("server.router()")),
				}
			}

			return &object.Struct{
				TypeName: "Router",
				Mutable:  true,
				Fields: map[string]object.Object{
					"routes": &object.Array{
						Elements:    []object.Object{},
						Mutable:     true,
						ElementType: "Route",
					},
					"middleware": &object.Array{
						Elements:    []object.Object{},
						Mutable:     true,
						ElementType: "func",
					},
				},
			}
		},
	},

	// route adds a route to an existing Router.
	// Supports path parameters: "/users/:id" — matched segments populate req.params
	"server.route": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 4 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 4 arguments", errors.Ident("server.route()")),
				}
			}

			router, ok := args[0].(*object.Struct)
			if !ok || router.TypeName != "Router" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the first argument", errors.Ident("server.route()"), errors.TypeExpected("Router")),
				}
			}

			method, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.route()"), errors.TypeExpected("string")),
				}
			}

			path, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the third argument", errors.Ident("server.route()"), errors.TypeExpected("string")),
				}
			}

			handler, ok := args[3].(*object.Function)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a function reference as the fourth argument (use ()handler_name)", errors.Ident("server.route()")),
				}
			}

			route := &object.Struct{
				TypeName: "Route",
				Mutable:  false,
				Fields: map[string]object.Object{
					"method":  method,
					"path":    path,
					"handler": handler,
				},
			}

			routes := router.Fields["routes"].(*object.Array)
			routes.Elements = append(routes.Elements, route)

			return &object.Nil{}
		},
	},

	// use adds middleware to the router.
	// Middleware functions receive (req Request) and return a Response or nil.
	// If middleware returns a Response, the chain stops and that response is sent.
	// If middleware returns nil, the next middleware/handler runs.
	"server.use": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("server.use()")),
				}
			}

			router, ok := args[0].(*object.Struct)
			if !ok || router.TypeName != "Router" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the first argument", errors.Ident("server.use()"), errors.TypeExpected("Router")),
				}
			}

			middleware, ok := args[1].(*object.Function)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a function reference as the second argument", errors.Ident("server.use()")),
				}
			}

			mw := router.Fields["middleware"].(*object.Array)
			mw.Elements = append(mw.Elements, middleware)

			return &object.Nil{}
		},
	},

	// static serves static files from a directory.
	// server.static(router, "/public", "./static")
	"server.static": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 3 arguments", errors.Ident("server.static()")),
				}
			}

			router, ok := args[0].(*object.Struct)
			if !ok || router.TypeName != "Router" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the first argument", errors.Ident("server.static()"), errors.TypeExpected("Router")),
				}
			}

			urlPrefix, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.static()"), errors.TypeExpected("string")),
				}
			}

			dirPath, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the third argument", errors.Ident("server.static()"), errors.TypeExpected("string")),
				}
			}

			// Store as a special static route
			route := &object.Struct{
				TypeName: "Route",
				Mutable:  false,
				Fields: map[string]object.Object{
					"method":     &object.String{Value: "GET"},
					"path":       urlPrefix,
					"handler":    &object.Nil{},
					"static_dir": dirPath,
				},
			}

			routes := router.Fields["routes"].(*object.Array)
			routes.Elements = append(routes.Elements, route)

			return &object.Nil{}
		},
	},

	// parse_json parses the request body as JSON and returns a map.
	"server.parse_json": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 1 argument", errors.Ident("server.parse_json()")),
				}
			}

			req, ok := args[0].(*object.Struct)
			if !ok || req.TypeName != "Request" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the argument", errors.Ident("server.parse_json()"), errors.TypeExpected("Request")),
				}
			}

			bodyObj, ok := req.Fields["body"].(*object.String)
			if !ok || bodyObj.Value == "" {
				return &object.Nil{}
			}

			var raw map[string]interface{}
			if err := json.Unmarshal([]byte(bodyObj.Value), &raw); err != nil {
				return &object.Error{
					Code:    errors.E18003.Code,
					Message: fmt.Sprintf("failed to parse JSON body: %s", err.Error()),
				}
			}

			return jsonToEZMap(raw)
		},
	},

	// set_header returns a new Response with an additional header set.
	"server.set_header": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 3 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 3 arguments", errors.Ident("server.set_header()")),
				}
			}

			resp, ok := args[0].(*object.Struct)
			if !ok || resp.TypeName != "Response" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the first argument", errors.Ident("server.set_header()"), errors.TypeExpected("Response")),
				}
			}

			key, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.set_header()"), errors.TypeExpected("string")),
				}
			}

			val, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the third argument", errors.Ident("server.set_header()"), errors.TypeExpected("string")),
				}
			}

			headers := resp.Fields["headers"].(*object.Map)
			headers.Set(&object.String{Value: strings.ToLower(key.Value)}, val)

			return resp
		},
	},

	// redirect creates a redirect Response.
	"server.redirect": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("server.redirect()")),
				}
			}

			status, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires an %s as the first argument", errors.Ident("server.redirect()"), errors.TypeExpected("int")),
				}
			}

			url, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.redirect()"), errors.TypeExpected("string")),
				}
			}

			resp := newServerResponse(int(status.Value.Int64()), "", "text/plain")
			headers := resp.Fields["headers"].(*object.Map)
			headers.Set(&object.String{Value: "location"}, url)
			return resp
		},
	},

	// cors adds CORS headers to the router via middleware.
	// server.cors(router, "*") or server.cors(router, "https://example.com")
	"server.cors": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("server.cors()")),
				}
			}

			router, ok := args[0].(*object.Struct)
			if !ok || router.TypeName != "Router" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the first argument", errors.Ident("server.cors()"), errors.TypeExpected("Router")),
				}
			}

			origin, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.cors()"), errors.TypeExpected("string")),
				}
			}

			// Store CORS origin on the router for the listen handler to use
			router.Fields["cors_origin"] = origin

			return &object.Nil{}
		},
	},

	// listen starts an HTTP server on the given port using the given router.
	"server.listen": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("server.listen()")),
				}
			}

			portArg, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires an %s as the first argument", errors.Ident("server.listen()"), errors.TypeExpected("int")),
				}
			}

			port := portArg.Value.Int64()
			if port < 1 || port > 65535 {
				return &object.Error{
					Code:    errors.E18002.Code,
					Message: fmt.Sprintf("port must be between 1 and 65535, got %d", port),
				}
			}

			router, ok := args[1].(*object.Struct)
			if !ok || router.TypeName != "Router" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.listen()"), errors.TypeExpected("Router")),
				}
			}

			routes, ok := router.Fields["routes"].(*object.Array)
			if !ok {
				return &object.Error{
					Code:    errors.E18003.Code,
					Message: "router has invalid routes field",
				}
			}

			middleware, _ := router.Fields["middleware"].(*object.Array)

			// Check for CORS origin
			var corsOrigin string
			if originObj, ok := router.Fields["cors_origin"].(*object.String); ok {
				corsOrigin = originObj.Value
			}

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				// Apply CORS headers if configured
				if corsOrigin != "" {
					w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
					if r.Method == "OPTIONS" {
						w.WriteHeader(http.StatusNoContent)
						return
					}
				}

				// Build the request struct
				request := buildRequest(r)

				// Run middleware chain
				if middleware != nil {
					for _, mwElem := range middleware.Elements {
						mwFn, ok := mwElem.(*object.Function)
						if !ok || mwFn.Call == nil {
							continue
						}
						result := mwFn.Call([]object.Object{request})
						// If middleware returns a Response, short-circuit
						if resp, ok := result.(*object.Struct); ok && resp.TypeName == "Response" {
							writeHTTPResponse(w, resp)
							return
						}
						if rv, ok := result.(*object.ReturnValue); ok && len(rv.Values) > 0 {
							if resp, ok := rv.Values[0].(*object.Struct); ok && resp.TypeName == "Response" {
								writeHTTPResponse(w, resp)
								return
							}
						}
						// nil return = continue to next middleware/handler
					}
				}

				// Match routes
				for _, elem := range routes.Elements {
					route, ok := elem.(*object.Struct)
					if !ok || route.TypeName != "Route" {
						continue
					}

					routeMethod, _ := route.Fields["method"].(*object.String)
					routePath, _ := route.Fields["path"].(*object.String)

					if routeMethod == nil || routePath == nil {
						continue
					}

					if r.Method != routeMethod.Value {
						continue
					}

					// Check for static file route
					if staticDir, ok := route.Fields["static_dir"].(*object.String); ok {
						prefix := routePath.Value
						if strings.HasPrefix(r.URL.Path, prefix) {
							serveStaticFile(w, r, prefix, staticDir.Value)
							return
						}
						continue
					}

					// Match path with parameters
					params, matched := matchPath(routePath.Value, r.URL.Path)
					if !matched {
						continue
					}

					// Set params on request
					request.Fields["params"] = params

					handler, _ := route.Fields["handler"].(*object.Function)
					if handler == nil || handler.Call == nil {
						continue
					}

					result := handler.Call([]object.Object{request})
					if resp, ok := result.(*object.Struct); ok && resp.TypeName == "Response" {
						writeHTTPResponse(w, resp)
						return
					}
					if rv, ok := result.(*object.ReturnValue); ok && len(rv.Values) > 0 {
						if resp, ok := rv.Values[0].(*object.Struct); ok && resp.TypeName == "Response" {
							writeHTTPResponse(w, resp)
							return
						}
					}
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					return
				}

				http.Error(w, "Not Found", http.StatusNotFound)
			})

			addr := fmt.Sprintf(":%d", port)
			err := http.ListenAndServe(addr, mux)
			if err != nil {
				return CreateStdlibError(errors.E18001.Code, err.Error())
			}
			return &object.Nil{}
		},
	},

	// text creates a Response struct with text/plain Content-Type.
	"server.text": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("server.text()")),
				}
			}
			status, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires an %s as the first argument", errors.Ident("server.text()"), errors.TypeExpected("int")),
				}
			}
			body, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.text()"), errors.TypeExpected("string")),
				}
			}
			return newServerResponse(int(status.Value.Int64()), body.Value, "text/plain")
		},
	},

	// json creates a Response struct with application/json Content-Type.
	"server.json": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("server.json()")),
				}
			}
			status, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires an %s as the first argument", errors.Ident("server.json()"), errors.TypeExpected("int")),
				}
			}
			jsonStr, jsonErr := encodeToJSON(args[1], make(map[uintptr]bool))
			if jsonErr != nil {
				return &object.Error{
					Code:    errors.E18003.Code,
					Message: fmt.Sprintf("failed to encode response body to JSON: %s", jsonErr.message),
				}
			}
			return newServerResponse(int(status.Value.Int64()), jsonStr, "application/json")
		},
	},

	// html creates a Response struct with text/html Content-Type.
	"server.html": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes exactly 2 arguments", errors.Ident("server.html()")),
				}
			}
			status, ok := args[0].(*object.Integer)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires an %s as the first argument", errors.Ident("server.html()"), errors.TypeExpected("int")),
				}
			}
			body, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the second argument", errors.Ident("server.html()"), errors.TypeExpected("string")),
				}
			}
			return newServerResponse(int(status.Value.Int64()), body.Value, "text/html")
		},
	},
}

// ============================================================================
// Helpers
// ============================================================================

func newServerResponse(status int, body string, contentType string) *object.Struct {
	headers := object.NewMap()
	headers.KeyType = "string"
	headers.ValueType = "string"
	headers.Set(&object.String{Value: "content-type"}, &object.String{Value: contentType})

	return &object.Struct{
		TypeName: "Response",
		Mutable:  false,
		Fields: map[string]object.Object{
			"status":  &object.Integer{Value: big.NewInt(int64(status))},
			"body":    &object.String{Value: body},
			"headers": headers,
		},
	}
}

func buildRequest(r *http.Request) *object.Struct {
	query := object.NewMap()
	query.KeyType = "string"
	query.ValueType = "string"
	for key, values := range r.URL.Query() {
		query.Set(&object.String{Value: key}, &object.String{Value: strings.Join(values, ",")})
	}

	headers := object.NewMap()
	headers.KeyType = "string"
	headers.ValueType = "string"
	for key, values := range r.Header {
		headers.Set(&object.String{Value: strings.ToLower(key)}, &object.String{Value: strings.Join(values, ",")})
	}

	params := object.NewMap()
	params.KeyType = "string"
	params.ValueType = "string"

	bodyStr := ""
	if r.Body != nil {
		bodyBytes, err := io.ReadAll(r.Body)
		if err == nil {
			bodyStr = string(bodyBytes)
		}
		r.Body.Close()
	}

	return &object.Struct{
		TypeName: "Request",
		Mutable:  true,
		Fields: map[string]object.Object{
			"method":  &object.String{Value: r.Method},
			"path":    &object.String{Value: r.URL.Path},
			"query":   query,
			"headers": headers,
			"params":  params,
			"body":    &object.String{Value: bodyStr},
		},
	}
}

// matchPath matches a route pattern against a request path.
// Supports :param segments. Returns (params map, matched bool).
func matchPath(pattern, requestPath string) (*object.Map, bool) {
	params := object.NewMap()
	params.KeyType = "string"
	params.ValueType = "string"

	patternParts := strings.Split(strings.Trim(pattern, "/"), "/")
	requestParts := strings.Split(strings.Trim(requestPath, "/"), "/")

	if len(patternParts) != len(requestParts) {
		return params, false
	}

	for i, part := range patternParts {
		if strings.HasPrefix(part, ":") {
			// Parameter segment — capture value
			paramName := part[1:]
			params.Set(&object.String{Value: paramName}, &object.String{Value: requestParts[i]})
		} else if part != requestParts[i] {
			return params, false
		}
	}

	return params, true
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, prefix, dir string) {
	// Strip the prefix to get the relative file path
	relPath := strings.TrimPrefix(r.URL.Path, prefix)
	relPath = strings.TrimPrefix(relPath, "/")

	if relPath == "" {
		relPath = "index.html"
	}

	// Prevent directory traversal
	cleanPath := filepath.Clean(relPath)
	if strings.Contains(cleanPath, "..") {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	fullPath := filepath.Join(dir, cleanPath)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err != nil || info.IsDir() {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, fullPath)
}

func writeHTTPResponse(w http.ResponseWriter, resp *object.Struct) {
	statusCode := http.StatusOK
	if statusObj, ok := resp.Fields["status"].(*object.Integer); ok {
		statusCode = int(statusObj.Value.Int64())
	}

	if headersObj, ok := resp.Fields["headers"].(*object.Map); ok {
		for _, pair := range headersObj.Pairs {
			key, keyOk := pair.Key.(*object.String)
			val, valOk := pair.Value.(*object.String)
			if keyOk && valOk {
				w.Header().Set(key.Value, val.Value)
			}
		}
	}

	bodyStr := ""
	if bodyObj, ok := resp.Fields["body"].(*object.String); ok {
		bodyStr = bodyObj.Value
	}

	w.WriteHeader(statusCode)
	fmt.Fprint(w, bodyStr)
}

// jsonToEZMap converts a Go map from json.Unmarshal to an EZ Map object.
func jsonToEZMap(raw map[string]interface{}) *object.Map {
	m := object.NewMap()
	m.KeyType = "string"
	m.ValueType = "any"

	for key, val := range raw {
		k := &object.String{Value: key}
		switch v := val.(type) {
		case string:
			m.Set(k, &object.String{Value: v})
		case float64:
			if v == float64(int64(v)) {
				m.Set(k, &object.Integer{Value: big.NewInt(int64(v))})
			} else {
				m.Set(k, &object.Float{Value: v})
			}
		case bool:
			m.Set(k, &object.Boolean{Value: v})
		case nil:
			m.Set(k, &object.Nil{})
		case map[string]interface{}:
			m.Set(k, jsonToEZMap(v))
		default:
			m.Set(k, &object.String{Value: fmt.Sprintf("%v", v)})
		}
	}

	return m
}

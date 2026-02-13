package stdlib

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

var ServerBuiltins = map[string]*object.Builtin{
	// router creates a new empty Router struct.
	// Returns a Router with an empty routes array.
	"server.router": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 0 {
				return &object.Error{
					Code:    "E7001",
					Message: fmt.Sprintf("%s takes no arguments", errors.Ident("server.router()")),
				}
			}

			routes := &object.Array{
				Elements:    []object.Object{},
				Mutable:     true,
				ElementType: "Route",
			}

			return &object.Struct{
				TypeName: "Router",
				Mutable:  true,
				Fields: map[string]object.Object{
					"routes": routes,
				},
			}
		},
	},

	// route adds a route to an existing Router.
	// Takes (router Router, method string, path string, response Response).
	// Mutates the router's routes array in place. Returns nil.
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

			response, ok := args[3].(*object.Struct)
			if !ok || response.TypeName != "Response" {
				return &object.Error{
					Code:    "E7003",
					Message: fmt.Sprintf("%s requires a %s as the fourth argument", errors.Ident("server.route()"), errors.TypeExpected("Response")),
				}
			}

			route := &object.Struct{
				TypeName: "Route",
				Mutable:  false,
				Fields: map[string]object.Object{
					"method":   method,
					"path":     path,
					"response": response,
				},
			}

			routes := router.Fields["routes"].(*object.Array)
			routes.Elements = append(routes.Elements, route)

			return &object.Nil{}
		},
	},

	// listen starts an HTTP server on the given port using the given router.
	// Blocks until the server encounters an error.
	// Returns Error on failure, nil on clean shutdown.
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

			mux := http.NewServeMux()
			mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				for _, elem := range routes.Elements {
					route, ok := elem.(*object.Struct)
					if !ok || route.TypeName != "Route" {
						continue
					}

					routeMethod, _ := route.Fields["method"].(*object.String)
					routePath, _ := route.Fields["path"].(*object.String)
					routeResp, _ := route.Fields["response"].(*object.Struct)

					if routeMethod == nil || routePath == nil || routeResp == nil {
						continue
					}

					if r.Method == routeMethod.Value && r.URL.Path == routePath.Value {
						writeHTTPResponse(w, routeResp)
						return
					}
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
	// Automatically encodes the data argument to JSON.
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

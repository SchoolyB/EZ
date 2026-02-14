package stdlib

import (
	"math/big"
	"net/http/httptest"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// Helper functions
// ============================================================================

func serverMakeInt(n int64) *object.Integer {
	return &object.Integer{Value: big.NewInt(n)}
}

func serverMakeStr(s string) *object.String {
	return &object.String{Value: s}
}

func serverGetResponseField(t *testing.T, resp *object.Struct, field string) object.Object {
	t.Helper()
	val, ok := resp.Fields[field]
	if !ok {
		t.Fatalf("Response missing field %q", field)
	}
	return val
}

func serverAssertResponseHeader(t *testing.T, resp *object.Struct, key, expected string) {
	t.Helper()
	headers, ok := resp.Fields["headers"].(*object.Map)
	if !ok {
		t.Fatal("Response headers is not a Map")
	}
	val, found := headers.Get(&object.String{Value: key})
	if !found {
		t.Fatalf("header %q not found", key)
	}
	str, ok := val.(*object.String)
	if !ok {
		t.Fatalf("header %q is not a string", key)
	}
	if str.Value != expected {
		t.Errorf("header %q = %q, want %q", key, str.Value, expected)
	}
}

// ============================================================================
// server.router tests
// ============================================================================

func TestServerRouter(t *testing.T) {
	fn := ServerBuiltins["server.router"].Fn

	result := fn()
	router, ok := result.(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct, got %T", result)
	}
	if router.TypeName != "Router" {
		t.Errorf("TypeName = %q, want %q", router.TypeName, "Router")
	}
	if !router.Mutable {
		t.Error("Router should be mutable")
	}

	routes, ok := router.Fields["routes"].(*object.Array)
	if !ok {
		t.Fatal("routes field is not an Array")
	}
	if len(routes.Elements) != 0 {
		t.Errorf("routes should be empty, got %d elements", len(routes.Elements))
	}
}

func TestServerRouterWrongArgCount(t *testing.T) {
	fn := ServerBuiltins["server.router"].Fn

	result := fn(serverMakeStr("extra"))
	errObj, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}
	if errObj.Code != "E7001" {
		t.Errorf("error code = %q, want %q", errObj.Code, "E7001")
	}
}

// ============================================================================
// server.route tests
// ============================================================================

func TestServerRoute(t *testing.T) {
	routerFn := ServerBuiltins["server.router"].Fn
	routeFn := ServerBuiltins["server.route"].Fn
	textFn := ServerBuiltins["server.text"].Fn

	router := routerFn()
	resp := textFn(serverMakeInt(200), serverMakeStr("hello"))

	result := routeFn(router, serverMakeStr("GET"), serverMakeStr("/"), resp)
	if _, ok := result.(*object.Nil); !ok {
		t.Fatalf("expected Nil return, got %T", result)
	}

	routes := router.(*object.Struct).Fields["routes"].(*object.Array)
	if len(routes.Elements) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes.Elements))
	}

	route := routes.Elements[0].(*object.Struct)
	if route.TypeName != "Route" {
		t.Errorf("route TypeName = %q, want %q", route.TypeName, "Route")
	}
	if route.Fields["method"].(*object.String).Value != "GET" {
		t.Error("route method should be GET")
	}
	if route.Fields["path"].(*object.String).Value != "/" {
		t.Error("route path should be /")
	}
}

func TestServerRouteMultiple(t *testing.T) {
	routerFn := ServerBuiltins["server.router"].Fn
	routeFn := ServerBuiltins["server.route"].Fn
	textFn := ServerBuiltins["server.text"].Fn

	router := routerFn()
	routeFn(router, serverMakeStr("GET"), serverMakeStr("/"), textFn(serverMakeInt(200), serverMakeStr("home")))
	routeFn(router, serverMakeStr("POST"), serverMakeStr("/api"), textFn(serverMakeInt(201), serverMakeStr("created")))
	routeFn(router, serverMakeStr("DELETE"), serverMakeStr("/item"), textFn(serverMakeInt(204), serverMakeStr("")))

	routes := router.(*object.Struct).Fields["routes"].(*object.Array)
	if len(routes.Elements) != 3 {
		t.Fatalf("expected 3 routes, got %d", len(routes.Elements))
	}

	methods := []string{"GET", "POST", "DELETE"}
	paths := []string{"/", "/api", "/item"}
	for i, elem := range routes.Elements {
		route := elem.(*object.Struct)
		if route.Fields["method"].(*object.String).Value != methods[i] {
			t.Errorf("route[%d] method = %q, want %q", i, route.Fields["method"].(*object.String).Value, methods[i])
		}
		if route.Fields["path"].(*object.String).Value != paths[i] {
			t.Errorf("route[%d] path = %q, want %q", i, route.Fields["path"].(*object.String).Value, paths[i])
		}
	}
}

func TestServerRouteWrongArgCount(t *testing.T) {
	fn := ServerBuiltins["server.route"].Fn

	for _, count := range []int{0, 1, 2, 3, 5} {
		args := make([]object.Object, count)
		for i := range args {
			args[i] = serverMakeStr("x")
		}
		result := fn(args...)
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Errorf("with %d args: expected Error, got %T", count, result)
			continue
		}
		if errObj.Code != "E7001" {
			t.Errorf("with %d args: error code = %q, want %q", count, errObj.Code, "E7001")
		}
	}
}

func TestServerRouteWrongArgTypes(t *testing.T) {
	routeFn := ServerBuiltins["server.route"].Fn
	textFn := ServerBuiltins["server.text"].Fn
	routerFn := ServerBuiltins["server.router"].Fn

	router := routerFn()
	resp := textFn(serverMakeInt(200), serverMakeStr("ok"))

	tests := []struct {
		name string
		args []object.Object
	}{
		{"wrong router type", []object.Object{serverMakeStr("not-router"), serverMakeStr("GET"), serverMakeStr("/"), resp}},
		{"wrong method type", []object.Object{router, serverMakeInt(1), serverMakeStr("/"), resp}},
		{"wrong path type", []object.Object{router, serverMakeStr("GET"), serverMakeInt(1), resp}},
		{"wrong response type", []object.Object{router, serverMakeStr("GET"), serverMakeStr("/"), serverMakeStr("not-resp")}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := routeFn(tt.args...)
			errObj, ok := result.(*object.Error)
			if !ok {
				t.Fatalf("expected Error, got %T", result)
			}
			if errObj.Code != "E7003" {
				t.Errorf("error code = %q, want %q", errObj.Code, "E7003")
			}
		})
	}
}

// ============================================================================
// server.text tests
// ============================================================================

func TestServerText(t *testing.T) {
	fn := ServerBuiltins["server.text"].Fn

	result := fn(serverMakeInt(200), serverMakeStr("hello"))
	resp, ok := result.(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct, got %T", result)
	}
	if resp.TypeName != "Response" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "Response")
	}
	if resp.Mutable {
		t.Error("Response should be immutable")
	}

	status := serverGetResponseField(t, resp, "status").(*object.Integer)
	if status.Value.Int64() != 200 {
		t.Errorf("status = %d, want 200", status.Value.Int64())
	}

	body := serverGetResponseField(t, resp, "body").(*object.String)
	if body.Value != "hello" {
		t.Errorf("body = %q, want %q", body.Value, "hello")
	}

	serverAssertResponseHeader(t, resp, "content-type", "text/plain")
}

func TestServerTextWrongArgCount(t *testing.T) {
	fn := ServerBuiltins["server.text"].Fn

	for _, args := range [][]object.Object{
		{},
		{serverMakeInt(200)},
		{serverMakeInt(200), serverMakeStr("a"), serverMakeStr("b")},
	} {
		result := fn(args...)
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Errorf("with %d args: expected Error, got %T", len(args), result)
			continue
		}
		if errObj.Code != "E7001" {
			t.Errorf("with %d args: error code = %q, want %q", len(args), errObj.Code, "E7001")
		}
	}
}

func TestServerTextWrongArgTypes(t *testing.T) {
	fn := ServerBuiltins["server.text"].Fn

	t.Run("wrong status type", func(t *testing.T) {
		result := fn(serverMakeStr("bad"), serverMakeStr("body"))
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E7003" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E7003")
		}
	})

	t.Run("wrong body type", func(t *testing.T) {
		result := fn(serverMakeInt(200), serverMakeInt(123))
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E7003" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E7003")
		}
	})
}

// ============================================================================
// server.json tests
// ============================================================================

func TestServerJson(t *testing.T) {
	fn := ServerBuiltins["server.json"].Fn

	m := object.NewMap()
	m.KeyType = "string"
	m.ValueType = "string"
	m.Set(serverMakeStr("status"), serverMakeStr("ok"))

	result := fn(serverMakeInt(200), m)
	resp, ok := result.(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct, got %T", result)
	}
	if resp.TypeName != "Response" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "Response")
	}

	body := serverGetResponseField(t, resp, "body").(*object.String)
	if body.Value != `{"status":"ok"}` {
		t.Errorf("body = %q, want %q", body.Value, `{"status":"ok"}`)
	}

	serverAssertResponseHeader(t, resp, "content-type", "application/json")
}

func TestServerJsonEncodesVariousTypes(t *testing.T) {
	fn := ServerBuiltins["server.json"].Fn

	tests := []struct {
		name     string
		input    object.Object
		expected string
	}{
		{"string", serverMakeStr("hello"), `"hello"`},
		{"integer", serverMakeInt(42), `42`},
		{"boolean true", &object.Boolean{Value: true}, `true`},
		{"boolean false", &object.Boolean{Value: false}, `false`},
		{"nil", &object.Nil{}, `null`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fn(serverMakeInt(200), tt.input)
			resp, ok := result.(*object.Struct)
			if !ok {
				t.Fatalf("expected Struct, got %T", result)
			}
			body := resp.Fields["body"].(*object.String)
			if body.Value != tt.expected {
				t.Errorf("body = %q, want %q", body.Value, tt.expected)
			}
		})
	}
}

func TestServerJsonWrongArgs(t *testing.T) {
	fn := ServerBuiltins["server.json"].Fn

	t.Run("wrong arg count", func(t *testing.T) {
		result := fn(serverMakeInt(200))
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E7001" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E7001")
		}
	})

	t.Run("wrong status type", func(t *testing.T) {
		result := fn(serverMakeStr("bad"), serverMakeStr("body"))
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E7003" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E7003")
		}
	})
}

// ============================================================================
// server.html tests
// ============================================================================

func TestServerHtml(t *testing.T) {
	fn := ServerBuiltins["server.html"].Fn

	result := fn(serverMakeInt(200), serverMakeStr("<h1>Hi</h1>"))
	resp, ok := result.(*object.Struct)
	if !ok {
		t.Fatalf("expected Struct, got %T", result)
	}
	if resp.TypeName != "Response" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "Response")
	}

	body := serverGetResponseField(t, resp, "body").(*object.String)
	if body.Value != "<h1>Hi</h1>" {
		t.Errorf("body = %q, want %q", body.Value, "<h1>Hi</h1>")
	}

	serverAssertResponseHeader(t, resp, "content-type", "text/html")
}

func TestServerHtmlWrongArgs(t *testing.T) {
	fn := ServerBuiltins["server.html"].Fn

	t.Run("wrong arg count", func(t *testing.T) {
		result := fn()
		if errObj, ok := result.(*object.Error); !ok || errObj.Code != "E7001" {
			t.Errorf("expected E7001, got %T %v", result, result)
		}
	})

	t.Run("wrong status type", func(t *testing.T) {
		result := fn(serverMakeStr("bad"), serverMakeStr("body"))
		if errObj, ok := result.(*object.Error); !ok || errObj.Code != "E7003" {
			t.Errorf("expected E7003, got %T %v", result, result)
		}
	})

	t.Run("wrong body type", func(t *testing.T) {
		result := fn(serverMakeInt(200), serverMakeInt(123))
		if errObj, ok := result.(*object.Error); !ok || errObj.Code != "E7003" {
			t.Errorf("expected E7003, got %T %v", result, result)
		}
	})
}

// ============================================================================
// server.listen validation tests (no actual server start)
// ============================================================================

func TestServerListenInvalidPort(t *testing.T) {
	fn := ServerBuiltins["server.listen"].Fn
	routerFn := ServerBuiltins["server.router"].Fn
	router := routerFn()

	t.Run("port 0", func(t *testing.T) {
		result := fn(serverMakeInt(0), router)
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E18002" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E18002")
		}
	})

	t.Run("port 99999", func(t *testing.T) {
		result := fn(serverMakeInt(99999), router)
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E18002" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E18002")
		}
	})
}

func TestServerListenWrongArgTypes(t *testing.T) {
	fn := ServerBuiltins["server.listen"].Fn
	routerFn := ServerBuiltins["server.router"].Fn
	router := routerFn()

	t.Run("wrong port type", func(t *testing.T) {
		result := fn(serverMakeStr("8080"), router)
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E7003" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E7003")
		}
	})

	t.Run("wrong router type", func(t *testing.T) {
		result := fn(serverMakeInt(8080), serverMakeStr("not-a-router"))
		errObj, ok := result.(*object.Error)
		if !ok {
			t.Fatalf("expected Error, got %T", result)
		}
		if errObj.Code != "E7003" {
			t.Errorf("error code = %q, want %q", errObj.Code, "E7003")
		}
	})
}

func TestServerListenWrongArgCount(t *testing.T) {
	fn := ServerBuiltins["server.listen"].Fn

	result := fn(serverMakeInt(8080))
	errObj, ok := result.(*object.Error)
	if !ok {
		t.Fatalf("expected Error, got %T", result)
	}
	if errObj.Code != "E7001" {
		t.Errorf("error code = %q, want %q", errObj.Code, "E7001")
	}
}

// ============================================================================
// Helper function tests
// ============================================================================

func TestNewServerResponse(t *testing.T) {
	resp := newServerResponse(201, "created", "text/plain")

	if resp.TypeName != "Response" {
		t.Errorf("TypeName = %q, want %q", resp.TypeName, "Response")
	}
	if resp.Mutable {
		t.Error("Response should be immutable")
	}

	status := resp.Fields["status"].(*object.Integer)
	if status.Value.Int64() != 201 {
		t.Errorf("status = %d, want 201", status.Value.Int64())
	}

	body := resp.Fields["body"].(*object.String)
	if body.Value != "created" {
		t.Errorf("body = %q, want %q", body.Value, "created")
	}

	serverAssertResponseHeader(t, resp, "content-type", "text/plain")
}

func TestWriteHTTPResponse(t *testing.T) {
	t.Run("text response", func(t *testing.T) {
		resp := newServerResponse(200, "OK", "text/plain")
		recorder := httptest.NewRecorder()
		writeHTTPResponse(recorder, resp)

		if recorder.Code != 200 {
			t.Errorf("status code = %d, want 200", recorder.Code)
		}
		if recorder.Body.String() != "OK" {
			t.Errorf("body = %q, want %q", recorder.Body.String(), "OK")
		}
		if ct := recorder.Header().Get("Content-Type"); ct != "text/plain" {
			t.Errorf("Content-Type = %q, want %q", ct, "text/plain")
		}
	})

	t.Run("json response", func(t *testing.T) {
		resp := newServerResponse(200, `{"a":1}`, "application/json")
		recorder := httptest.NewRecorder()
		writeHTTPResponse(recorder, resp)

		if recorder.Code != 200 {
			t.Errorf("status code = %d, want 200", recorder.Code)
		}
		if recorder.Body.String() != `{"a":1}` {
			t.Errorf("body = %q, want %q", recorder.Body.String(), `{"a":1}`)
		}
		if ct := recorder.Header().Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}
	})

	t.Run("404 response", func(t *testing.T) {
		resp := newServerResponse(404, "Not Found", "text/plain")
		recorder := httptest.NewRecorder()
		writeHTTPResponse(recorder, resp)

		if recorder.Code != 404 {
			t.Errorf("status code = %d, want 404", recorder.Code)
		}
		if recorder.Body.String() != "Not Found" {
			t.Errorf("body = %q, want %q", recorder.Body.String(), "Not Found")
		}
	})
}

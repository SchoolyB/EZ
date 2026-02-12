package stdlib

import (
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/marshallburns/ez/pkg/object"
)

// ============================================================================
// HTTP request tests
// ============================================================================

func TestHTTPRequests(t *testing.T) {
	getFn := HttpBuiltins["http.get"].Fn
	postFn := HttpBuiltins["http.post"].Fn
	putFn := HttpBuiltins["http.put"].Fn
	deleteFn := HttpBuiltins["http.delete"].Fn
	patchFn := HttpBuiltins["http.patch"].Fn
	advRequestFn := HttpBuiltins["http.request"].Fn

	serverState := map[string]string{}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		body, _ := io.ReadAll(r.Body)
		q := r.URL.Query()
		id := q.Get("id")

		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			val, ok := serverState[id]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Write([]byte(val))

		case http.MethodPost:
			serverState[id] = string(body)
			w.WriteHeader(http.StatusCreated)

		case http.MethodPut:
			if _, ok := serverState[id]; !ok {
				http.NotFound(w, r)
				return
			}
			serverState[id] = string(body)
			w.WriteHeader(http.StatusOK)

		case http.MethodPatch:
			serverState[id] += string(body)
			w.WriteHeader(http.StatusOK)

		case http.MethodDelete:
			delete(serverState, id)
			w.WriteHeader(http.StatusNoContent)

		case http.MethodOptions:
			w.Header().Set("Allow", "GET,POST,PUT,DELETE,PATCH,OPTIONS,HEAD")
			w.WriteHeader(http.StatusNoContent)

		case http.MethodHead:
			w.WriteHeader(http.StatusOK)

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	t.Run("POST request", func(t *testing.T) {
		resource := &object.String{Value: server.URL + "?id=1"}
		res := postFn(resource, &object.String{Value: "hello"})
		assertNoError(t, res)

		status := getReturnValues(t, res)[0].(*object.Struct).Fields["status"].(*object.Integer).Value.Uint64()
		if status != http.StatusCreated {
			t.Fatalf("expected %d, got %d", http.StatusCreated, status)
		}

		val, ok := serverState["1"]
		if !ok {
			t.Fatalf("expected key to be set")
		}
		if val != "hello" {
			t.Fatalf("expected 'hello', got %q", val)
		}
	})

	t.Run("GET request", func(t *testing.T) {
		resource := &object.String{Value: server.URL + "?id=1"}
		res := getFn(resource)
		assertNoError(t, res)

		response := getReturnValues(t, res)[0].(*object.Struct)
		status := response.Fields["status"].(*object.Integer).Value.Uint64()
		if status != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, status)
		}

		body := response.Fields["body"].(*object.String).Value
		if body != "hello" {
			t.Fatalf("expected 'hello', got %q", body)
		}
	})

	t.Run("PUT request", func(t *testing.T) {
		resource := &object.String{Value: server.URL + "?id=1"}
		res := putFn(resource, &object.String{Value: "world"})
		assertNoError(t, res)

		status := getReturnValues(t, res)[0].(*object.Struct).Fields["status"].(*object.Integer).Value.Uint64()
		if status != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, status)
		}

		val, ok := serverState["1"]
		if !ok {
			t.Fatalf("expected key to be set")
		}
		if val != "world" {
			t.Fatalf("expected 'world', got %q", val)
		}
	})

	t.Run("PATCH request", func(t *testing.T) {
		resource := &object.String{Value: server.URL + "?id=1"}
		res := patchFn(resource, &object.String{Value: "123"})
		assertNoError(t, res)

		status := getReturnValues(t, res)[0].(*object.Struct).Fields["status"].(*object.Integer).Value.Uint64()
		if status != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, status)
		}

		val, ok := serverState["1"]
		if !ok {
			t.Fatalf("expected key to be set")
		}
		if val != "world123" {
			t.Fatalf("expected key to be 'world123', got %q", val)
		}
	})

	t.Run("OPTIONS request", func(t *testing.T) {
		method := &object.String{Value: "OPTIONS"}
		resource := &object.String{Value: server.URL}
		res := advRequestFn(method, resource, &object.String{Value: ""}, object.NewMap(), &object.Integer{Value: big.NewInt(0)})
		assertNoError(t, res)

		response := getReturnValues(t, res)[0].(*object.Struct)
		status := response.Fields["status"].(*object.Integer).Value.Uint64()
		if status != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, status)
		}

		headerValue, ok := response.Fields["headers"].(*object.Map).Get(&object.String{Value: "Allow"})
		if !ok {
			t.Fatalf("expected 'Allow' header field to be set")
		}

		if headerValue.(*object.Array).Elements[0].(*object.String).Value != "GET,POST,PUT,DELETE,PATCH,OPTIONS,HEAD" {
			t.Fatalf("header content not set correctly")
		}
	})

	t.Run("HEAD request", func(t *testing.T) {
		method := &object.String{Value: "HEAD"}
		resource := &object.String{Value: server.URL}
		res := advRequestFn(method, resource, &object.String{Value: ""}, object.NewMap(), &object.Integer{Value: big.NewInt(0)})
		assertNoError(t, res)

		response := getReturnValues(t, res)[0].(*object.Struct)
		status := response.Fields["status"].(*object.Integer).Value.Uint64()
		if status != http.StatusOK {
			t.Fatalf("expected %d, got %d", http.StatusOK, status)
		}
	})

	t.Run("DELETE request", func(t *testing.T) {
		resource := &object.String{Value: server.URL + "?id=1"}
		res := deleteFn(resource)
		assertNoError(t, res)

		status := getReturnValues(t, res)[0].(*object.Struct).Fields["status"].(*object.Integer).Value.Uint64()
		if status != http.StatusNoContent {
			t.Fatalf("expected %d, got %d", http.StatusNoContent, status)
		}

		if _, ok := serverState["1"]; ok {
			t.Fatalf("expected key, value pair to be deleted")
		}
	})
}

// ============================================================================
// HTTP url utility tests
// ============================================================================

func TestURLUtility(t *testing.T) {
	buildQueryFn := HttpBuiltins["http.build_query"].Fn

	t.Run("build_query", func(t *testing.T) {
		input := object.NewMap()
		input.Set(&object.String{Value: "q"}, &object.String{Value: "hello world"})
		input.Set(&object.String{Value: "page"}, &object.String{Value: "1"})

		res := buildQueryFn(input)

		str, ok := res.(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", res)
		}

		expected := "page=1&q=hello+world"
		if str.Value != expected {
			t.Fatalf("expected %q, got %q", expected, str.Value)
		}
	})
}

// ============================================================================
// HTTP json helper tests
// ============================================================================

func TestJSONelper(t *testing.T) {
	jsonHelperFn := HttpBuiltins["http.json_body"].Fn

	t.Run("simple map", func(t *testing.T) {
		input := object.NewMap()
		input.Set(&object.String{Value: "name"}, &object.String{Value: "alice"})
		input.Set(&object.String{Value: "age"}, &object.Integer{Value: big.NewInt(30)})

		res := jsonHelperFn(input)

		str, ok := res.(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", res)
		}
		expected := `{"age":30,"name":"alice"}`

		if str.Value != expected {
			t.Fatalf("unexpected json output: %q", str.Value)
		}
	})

	t.Run("nested map", func(t *testing.T) {
		address := object.NewMap()
		address.Set(&object.String{Value: "city"}, &object.String{Value: "Paris"})
		address.Set(&object.String{Value: "zip"}, &object.Integer{Value: big.NewInt(75000)})

		input := object.NewMap()
		input.Set(&object.String{Value: "user"}, address)
		input.Set(&object.String{Value: "active"}, &object.Boolean{Value: true})

		res := jsonHelperFn(input)

		str, ok := res.(*object.String)
		if !ok {
			t.Fatalf("expected String, got %T", res)
		}

		var decoded map[string]any
		if err := json.Unmarshal([]byte(str.Value), &decoded); err != nil {
			t.Fatalf("invalid json output: %v", err)
		}

		if decoded["active"] != true {
			t.Fatalf("expected active=true, got %v", decoded["active"])
		}

		user, ok := decoded["user"].(map[string]any)
		if !ok {
			t.Fatalf("expected user object, got %T", decoded["user"])
		}

		if user["city"] != "Paris" {
			t.Fatalf("expected city=Paris, got %v", user["city"])
		}
	})
}

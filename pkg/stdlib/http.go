package stdlib

import (
	"bytes"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/marshallburns/ez/pkg/object"
)

const DEFAULT_TIMEOUT = 30
const (
	OK                    int64 = 200
	CREATED               int64 = 201
	ACCEPTED              int64 = 202
	NO_CONTENT            int64 = 204
	MOVED_PERMANENTLY     int64 = 301
	FOUND                 int64 = 302
	NOT_MODIFIED          int64 = 304
	TEMPORARY_REDIRECT    int64 = 307
	PERMANENT_REDIRECT    int64 = 308
	BAD_REQUEST           int64 = 400
	UNAUTHORIZED          int64 = 401
	PAYMENT_REQUIRED      int64 = 402
	FORBIDDEN             int64 = 403
	NOT_FOUND             int64 = 404
	METHOD_NOT_ALLOWED    int64 = 405
	CONFLICT              int64 = 409
	INTERNAL_SERVER_ERROR int64 = 500
	BAD_GATEWAY           int64 = 502
	SERVICE_UNAVAILABLE   int64 = 503
)

var defaultClient = &http.Client{
	Timeout: time.Duration(DEFAULT_TIMEOUT) * time.Second,
}

var HttpBuiltins = map[string]*object.Builtin{
	// get performs an HTTP GET request to the specified URL.
	// Takes URL string. Returns (HttpResponse, Error) tuple.
	"http.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.get() takes exactly 1 argument"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.get() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodGet, urlArg.Value, nil)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, string(body), headers),
					&object.Nil{},
				},
			}
		},
	},

	// post performs an HTTP POST request with a request body.
	// Takes URL string and body string. Returns (HttpResponse, Error) tuple.
	"http.post": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "http.post() takes exactly 2 arguments"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.post() requires a string argument"}
			}

			body, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.post() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodPost, urlArg.Value, bytes.NewBuffer([]byte(body.Value)))
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, string(responseBody), headers),
					&object.Nil{},
				},
			}
		},
	},

	// put performs an HTTP PUT request with a request body.
	// Takes URL string and body string. Returns (HttpResponse, Error) tuple.
	"http.put": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "http.put() takes exactly 2 arguments"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.put() requires a string argument"}
			}

			body, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.put() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodPut, urlArg.Value, bytes.NewBuffer([]byte(body.Value)))
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, string(responseBody), headers),
					&object.Nil{},
				},
			}
		},
	},

	// delete performs an HTTP DELETE request to the specified URL.
	// Takes URL string. Returns (HttpResponse, Error) tuple.
	"http.delete": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.delete() takes exactly 1 argument"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.delete() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodDelete, urlArg.Value, nil)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, string(body), headers),
					&object.Nil{},
				},
			}
		},
	},

	// patch performs an HTTP PATCH request with a request body.
	// Takes URL string and body string. Returns (HttpResponse, Error) tuple.
	"http.patch": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "http.patch() takes exactly 2 arguments"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.patch() requires a string argument"}
			}

			body, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.patch() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodPatch, urlArg.Value, bytes.NewBuffer([]byte(body.Value)))
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, string(responseBody), headers),
					&object.Nil{},
				},
			}
		},
	},

	// request performs a configurable HTTP request with custom method, headers, and timeout.
	// Takes method, URL, body, headers map, and timeout (seconds). Returns (HttpResponse, Error) tuple.
	"http.request": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 5 {
				return &object.Error{Code: "E7001", Message: "http.request() takes exactly 5 arguments"}
			}

			methodArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.request() requires a string argument"}
			}

			urlArg, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.request() requires a string argument"}
			}

			bodyArg, ok := args[2].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.request() requires a string argument"}
			}

			headersArg, ok := args[3].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "http.request() requires a map argument"}
			}

			timeoutArg, ok := args[4].(*object.Integer)
			if !ok {
				return &object.Error{Code: "E7004", Message: "http.request() requires a integer argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			timeout := DEFAULT_TIMEOUT
			if timeoutArg.Value.Uint64() > 0 {
				timeout = int(timeoutArg.Value.Uint64())
			}

			client := http.Client{
				Timeout: time.Duration(timeout) * time.Second,
			}

			var req *http.Request
			var err error
			switch methodArg.Value {
			case "GET":
				req, err = http.NewRequest(http.MethodGet, urlArg.Value, nil)
			case "POST":
				req, err = http.NewRequest(http.MethodPost, urlArg.Value, bytes.NewBuffer([]byte(bodyArg.Value)))
			case "PUT":
				req, err = http.NewRequest(http.MethodPut, urlArg.Value, bytes.NewBuffer([]byte(bodyArg.Value)))
			case "DELETE":
				req, err = http.NewRequest(http.MethodDelete, urlArg.Value, nil)
			case "PATCH":
				req, err = http.NewRequest(http.MethodPatch, urlArg.Value, bytes.NewBuffer([]byte(bodyArg.Value)))
			case "OPTIONS":
				req, err = http.NewRequest(http.MethodOptions, urlArg.Value, nil)
			case "HEAD":
				req, err = http.NewRequest(http.MethodHead, urlArg.Value, nil)
			default:
				return &object.Error{Code: "E14004", Message: "invalid HTTP method: `" + methodArg.Value + "`"}
			}
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			for _, pair := range headersArg.Pairs {
				req.Header.Add(pair.Key.(*object.String).Value, pair.Value.(*object.String).Value)
			}

			res, err := client.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, string(responseBody), headers),
					&object.Nil{},
				},
			}
		},
	},

	// build_query encodes a map as a URL query string.
	// Takes map[string:string]. Returns URL-encoded query string.
	"http.build_query": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.build_query() takes exactly 1 arguments"}
			}

			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "http.build_query() requires a map argument"}
			}

			q := url.Values{}
			for _, pair := range m.Pairs {
				key, ok := pair.Key.(*object.String)
				if !ok {
					return &object.Error{Code: "E7007", Message: "http.build_query() requires a map[string:string] argument"}
				}
				val, ok := pair.Value.(*object.String)
				if !ok {
					return &object.Error{Code: "E7007", Message: "http.build_query() requires a map[string:string] argument"}
				}
				q.Set(key.Value, val.Value)
			}

			return &object.String{Value: q.Encode()}
		},
	},

	// json_body encodes a map as a JSON string for use in request bodies.
	// Takes a map. Returns JSON string.
	"http.json_body": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.json_body() takes exactly 1 arguments"}
			}

			m, ok := args[0].(*object.Map)
			if !ok {
				return &object.Error{Code: "E7007", Message: "http.json_body() requires a map argument"}
			}

			result, err := encodeToJSON(m, make(map[uintptr]bool))
			if err != nil {
				return &object.Error{Code: "E14006", Message: err.message}
			}

			return &object.String{Value: result}
		},
	},

	// head performs an HTTP HEAD request to retrieve headers without body.
	// Takes URL string. Returns (HttpResponse, Error) tuple with empty body.
	"http.head": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.head() takes exactly 1 argument"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.head() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodHead, urlArg.Value, nil)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, "", headers),
					&object.Nil{},
				},
			}
		},
	},

	// options performs an HTTP OPTIONS request to query supported methods.
	// Takes URL string. Returns (HttpResponse, Error) tuple.
	"http.options": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.options() takes exactly 1 argument"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.options() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodOptions, urlArg.Value, nil)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			headers.KeyType = "string"
			headers.ValueType = "[string]"
			for key, vals := range res.Header {
				values := &object.Array{ElementType: "string"}
				for _, val := range vals {
					values.Elements = append(values.Elements, &object.String{Value: val})
				}
				headers.Set(&object.String{Value: key}, values)
			}

			return &object.ReturnValue{
				Values: []object.Object{
					newHttpResponse(res.StatusCode, string(body), headers),
					&object.Nil{},
				},
			}
		},
	},

	// download fetches a URL and saves the content to a file.
	// Takes URL string and file path. Returns (bytes_written, Error) tuple.
	"http.download": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 2 {
				return &object.Error{Code: "E7001", Message: "http.download() takes exactly 2 arguments (url, path)"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.download() requires a string url as first argument"}
			}

			pathArg, ok := args[1].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.download() requires a string path as second argument"}
			}

			if _, err := url.ParseRequestURI(urlArg.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodGet, urlArg.Value, nil)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Integer{Value: big.NewInt(0)},
						CreateStdlibError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Integer{Value: big.NewInt(0)},
						CreateStdlibError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			file, err := createFile(pathArg.Value)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Integer{Value: big.NewInt(0)},
						CreateStdlibError("E14003", "could not create file: "+err.Error()),
					},
				}
			}
			defer file.Close()

			written, err := io.Copy(file, res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Integer{Value: big.NewInt(0)},
						CreateStdlibError("E14003", "could not write file: "+err.Error()),
					},
				}
			}

			return &object.ReturnValue{
				Values: []object.Object{
					&object.Integer{Value: big.NewInt(written)},
					&object.Nil{},
				},
			}
		},
	},

	// parse_url parses a URL string into its component parts.
	// Takes URL string. Returns (URL struct, Error) tuple.
	"http.parse_url": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.parse_url() takes exactly 1 argument"}
			}

			urlArg, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.parse_url() requires a string argument"}
			}

			parsed, err := url.Parse(urlArg.Value)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						CreateStdlibError("E14001", "invalid url: "+err.Error()),
					},
				}
			}

			port := parsed.Port()
			var portInt int64 = 0
			if port != "" {
				if p, err := parsePort(port); err == nil {
					portInt = p
				}
			}

			return &object.ReturnValue{
				Values: []object.Object{
					&object.Struct{
						TypeName: "URL",
						Mutable:  false,
						Fields: map[string]object.Object{
							"scheme":   &object.String{Value: parsed.Scheme},
							"host":     &object.String{Value: parsed.Hostname()},
							"port":     &object.Integer{Value: big.NewInt(portInt)},
							"path":     &object.String{Value: parsed.Path},
							"query":    &object.String{Value: parsed.RawQuery},
							"fragment": &object.String{Value: parsed.Fragment},
						},
					},
					&object.Nil{},
				},
			}
		},
	},

	// build_url constructs a URL string from component parts.
	// Takes URL struct with scheme, host, port, path, query, fragment. Returns URL string.
	"http.build_url": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.build_url() takes exactly 1 argument"}
			}

			components, ok := args[0].(*object.Struct)
			if !ok {
				return &object.Error{Code: "E7001", Message: "http.build_url() requires a struct argument"}
			}

			scheme := ""
			if s, ok := components.Fields["scheme"].(*object.String); ok {
				scheme = s.Value
			}

			host := ""
			if h, ok := components.Fields["host"].(*object.String); ok {
				host = h.Value
			}

			port := ""
			if p, ok := components.Fields["port"].(*object.Integer); ok && p.Value.Int64() > 0 {
				port = p.Value.String()
			}

			path := ""
			if pa, ok := components.Fields["path"].(*object.String); ok {
				path = pa.Value
			}

			query := ""
			if q, ok := components.Fields["query"].(*object.String); ok {
				query = q.Value
			}

			fragment := ""
			if f, ok := components.Fields["fragment"].(*object.String); ok {
				fragment = f.Value
			}

			u := &url.URL{
				Scheme:   scheme,
				Host:     host,
				Path:     path,
				RawQuery: query,
				Fragment: fragment,
			}

			if port != "" {
				u.Host = host + ":" + port
			}

			return &object.String{Value: u.String()}
		},
	},

	// ============================================================================
	// HTTP Status Code Constants
	// These constants provide standard HTTP status codes for response handling.
	// ============================================================================

	// OK is HTTP 200 status code indicating successful request.
	"http.OK": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(OK)}
		},
		IsConstant: true,
	},

	// CREATED is HTTP 201 status code indicating resource was created.
	"http.CREATED": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(CREATED)}
		},
		IsConstant: true,
	},

	// ACCEPTED is HTTP 202 status code indicating request accepted for processing.
	"http.ACCEPTED": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(ACCEPTED)}
		},
		IsConstant: true,
	},

	// NO_CONTENT is HTTP 204 status code indicating success with no response body.
	"http.NO_CONTENT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(NO_CONTENT)}
		},
		IsConstant: true,
	},

	// MOVED_PERMANENTLY is HTTP 301 status code indicating permanent redirect.
	"http.MOVED_PERMANENTLY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(MOVED_PERMANENTLY)}
		},
		IsConstant: true,
	},

	// FOUND is HTTP 302 status code indicating temporary redirect.
	"http.FOUND": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(FOUND)}
		},
		IsConstant: true,
	},

	// NOT_MODIFIED is HTTP 304 status code indicating cached version is current.
	"http.NOT_MODIFIED": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(NOT_MODIFIED)}
		},
		IsConstant: true,
	},

	// TEMPORARY_REDIRECT is HTTP 307 status code indicating temporary redirect preserving method.
	"http.TEMPORARY_REDIRECT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(TEMPORARY_REDIRECT)}
		},
		IsConstant: true,
	},

	// PERMANENT_REDIRECT is HTTP 308 status code indicating permanent redirect preserving method.
	"http.PERMANENT_REDIRECT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(PERMANENT_REDIRECT)}
		},
		IsConstant: true,
	},

	// BAD_REQUEST is HTTP 400 status code indicating malformed request.
	"http.BAD_REQUEST": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(BAD_REQUEST)}
		},
		IsConstant: true,
	},

	// UNAUTHORIZED is HTTP 401 status code indicating authentication required.
	"http.UNAUTHORIZED": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(UNAUTHORIZED)}
		},
		IsConstant: true,
	},

	// PAYMENT_REQUIRED is HTTP 402 status code reserved for future use.
	"http.PAYMENT_REQUIRED": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(PAYMENT_REQUIRED)}
		},
		IsConstant: true,
	},

	// FORBIDDEN is HTTP 403 status code indicating access denied.
	"http.FORBIDDEN": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(FORBIDDEN)}
		},
		IsConstant: true,
	},

	// NOT_FOUND is HTTP 404 status code indicating resource not found.
	"http.NOT_FOUND": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(NOT_FOUND)}
		},
		IsConstant: true,
	},

	// METHOD_NOT_ALLOWED is HTTP 405 status code indicating method not supported.
	"http.METHOD_NOT_ALLOWED": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(METHOD_NOT_ALLOWED)}
		},
		IsConstant: true,
	},

	// CONFLICT is HTTP 409 status code indicating request conflicts with server state.
	"http.CONFLICT": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(CONFLICT)}
		},
		IsConstant: true,
	},

	// INTERNAL_SERVER_ERROR is HTTP 500 status code indicating server error.
	"http.INTERNAL_SERVER_ERROR": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(INTERNAL_SERVER_ERROR)}
		},
		IsConstant: true,
	},

	// BAD_GATEWAY is HTTP 502 status code indicating invalid upstream response.
	"http.BAD_GATEWAY": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(BAD_GATEWAY)}
		},
		IsConstant: true,
	},

	// SERVICE_UNAVAILABLE is HTTP 503 status code indicating server temporarily unavailable.
	"http.SERVICE_UNAVAILABLE": {
		Fn: func(args ...object.Object) object.Object {
			return &object.Integer{Value: big.NewInt(SERVICE_UNAVAILABLE)}
		},
		IsConstant: true,
	},
}

func newHttpResponse(status int, body string, headers *object.Map) *object.Struct {
	return &object.Struct{
		TypeName: "HttpResponse",
		Mutable:  false,
		Fields: map[string]object.Object{
			"status":  &object.Integer{Value: big.NewInt(int64(status))},
			"body":    &object.String{Value: body},
			"headers": headers,
		},
	}
}

func createFile(path string) (*os.File, error) {
	return os.Create(path)
}

func parsePort(port string) (int64, error) {
	p, err := strconv.ParseInt(port, 10, 64)
	if err != nil {
		return 0, err
	}
	return p, nil
}

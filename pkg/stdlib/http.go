package stdlib

import (
	"bytes"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"time"

	"github.com/marshallburns/ez/pkg/object"
)

const DEFAULT_TIMEOUT = 30

var defaultClient = &http.Client{
	Timeout: time.Duration(DEFAULT_TIMEOUT)*time.Second,
}

var HttpBuiltins = map[string]*object.Builtin{
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
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			for key, vals := range res.Header {
				values := &object.Array{}
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
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			for key, vals := range res.Header {
				values := &object.Array{}
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
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			for key, vals := range res.Header {
				values := &object.Array{}
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
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			for key, vals := range res.Header {
				values := &object.Array{}
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
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := defaultClient.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			for key, vals := range res.Header {
				values := &object.Array{}
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
				Timeout: time.Duration(timeout)*time.Second,
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
				return &object.Error{Code: "E14004", Message: "invalid HTTP method: `"+methodArg.Value+"`"}
			}
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
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
						createHttpError("E14002", "request failed: "+err.Error()),
					},
				}
			}
			defer res.Body.Close()

			responseBody, err := io.ReadAll(res.Body)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "could not read body"),
					},
				}
			}

			headers := object.NewMap()
			for key, vals := range res.Header {
				values := &object.Array{}
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

	"http.encode_url":  {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.encode_url() takes exactly 1 arguments"}
			}

			s, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.encode_url() requires a string argument"}
			}

			encoded := url.QueryEscape(s.Value)

			return &object.String{Value: encoded}
		},
	},

	"http.decode_url":  {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.decode_url() takes exactly 1 arguments"}
			}

			s, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.decode_url() requires a string argument"}
			}

			decoded, err := url.QueryUnescape(s.Value)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14005", "URL decode failed"),
					},
				}
			}

			return &object.ReturnValue{
				Values: []object.Object{
					&object.String{Value: decoded},
					&object.Nil{},
				},
			}
		},
	},

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
} 

func createHttpError(code, message string) *object.Struct {
	return &object.Struct{
		TypeName: "Error",
		Mutable: false,
		Fields: map[string]object.Object{
			"message": &object.String{Value: message},
			"code":    &object.String{Value: code},
		},
	}
}

func newHttpResponse(status int, body string, headers *object.Map) *object.Struct {
	return &object.Struct{
		TypeName: "HttpResponse",
		Mutable: false,
		Fields: map[string]object.Object{
			"status": &object.Integer{Value: big.NewInt(int64(status))},
			"body": &object.String{Value: body},
			"headers": headers,
		},
	}
}

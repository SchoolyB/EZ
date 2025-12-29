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

var client = &http.Client{Timeout: time.Duration(30)*time.Second}

var HttpBuiltins = map[string]*object.Builtin{
	"http.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.get() takes exactly 1 argument"}
			}

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.get() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(str.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			req, err := http.NewRequest(http.MethodGet, str.Value, nil)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := client.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
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

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.post() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(str.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			body, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.post() requires a string argument"}
			}

			req, err := http.NewRequest(http.MethodPost, str.Value, bytes.NewBuffer([]byte(body.Value)))
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := client.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
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

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.put() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(str.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}

			body, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.put() requires a string argument"}
			}

			req, err := http.NewRequest(http.MethodPut, str.Value, bytes.NewBuffer([]byte(body.Value)))
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := client.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
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

			str, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.delete() requires a string argument"}
			}

			if _, err := url.ParseRequestURI(str.Value); err != nil {
				return &object.Error{Code: "E14001", Message: "invalid url"}
			}
			
			req, err := http.NewRequest(http.MethodDelete, str.Value, nil)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
					},
				}
			}

			res, err := client.Do(req)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
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

package stdlib

import (
	"io"
	"math/big"
	"net/http"

	"github.com/marshallburns/ez/pkg/object"
)

var HttpBuiltins = map[string]*object.Builtin{
	"http.get": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "http.get() takes exactly 1 argument"}
			}

			url, ok := args[0].(*object.String)
			if !ok {
				return &object.Error{Code: "E7003", Message: "http.get() requires a string argument"}
			}

			res, err := http.Get(url.Value)
			if err != nil {
				return &object.ReturnValue{
					Values: []object.Object{
						&object.Nil{},
						createHttpError("E14002", "request failed"),
					},
				}
			}

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

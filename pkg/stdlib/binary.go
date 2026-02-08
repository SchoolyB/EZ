package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"github.com/marshallburns/ez/pkg/object"
)

// Helper to convert EZ byte array to Go []byte with length validation
func binaryBytesToSlice(arg object.Object, expected int, funcName string) ([]byte, *object.Struct) {
	arr, ok := arg.(*object.Array)
	if !ok {
		return nil, CreateStdlibError("E7002", fmt.Sprintf("%s requires a byte array argument", funcName))
	}
	if len(arr.Elements) != expected {
		return nil, CreateStdlibError("E7010", fmt.Sprintf("%s requires exactly %d bytes, got %d", funcName, expected, len(arr.Elements)))
	}

	data := make([]byte, expected)
	for i, elem := range arr.Elements {
		switch e := elem.(type) {
		case *object.Byte:
			data[i] = e.Value
		case *object.Integer:
			val := e.Value.Int64()
			if val < 0 || val > 255 {
				return nil, CreateStdlibError("E3022", fmt.Sprintf("%s byte at index %d out of range (0-255)", funcName, i))
			}
			data[i] = byte(val)
		default:
			return nil, CreateStdlibError("E7002", fmt.Sprintf("%s requires a byte array", funcName))
		}
	}
	return data, nil
}

// Helper to convert Go []byte to EZ byte array
func sliceToBinaryArray(data []byte) *object.Array {
	elements := make([]object.Object, len(data))
	for i, b := range data {
		elements[i] = &object.Byte{Value: b}
	}
	return &object.Array{Elements: elements, ElementType: "byte"}
}

// Helper to extract integer value for encoding
func extractEncodingInt(arg object.Object, funcName string) (*big.Int, *object.Error) {
	switch v := arg.(type) {
	case *object.Integer:
		return v.Value, nil
	case *object.Float:
		return big.NewInt(int64(v.Value)), nil
	case *object.Byte:
		return big.NewInt(int64(v.Value)), nil
	default:
		return nil, &object.Error{
			Code:    "E7004",
			Message: fmt.Sprintf("%s requires an integer argument", funcName),
		}
	}
}

// Helper to reverse byte slice (for endianness conversion with big.Int)
func reverseBytes(data []byte) []byte {
	n := len(data)
	result := make([]byte, n)
	for i := 0; i < n; i++ {
		result[i] = data[n-1-i]
	}
	return result
}

// Helper to pad big.Int bytes to exact size (big-endian, for signed integers)
// Converts negative numbers to two's complement representation
func padBigIntBytesSigned(val *big.Int, size int) []byte {
	if val.Sign() >= 0 {
		// Positive: just pad with zeros
		bytes := val.Bytes()
		if len(bytes) > size {
			return bytes[len(bytes)-size:]
		}
		result := make([]byte, size)
		copy(result[size-len(bytes):], bytes)
		return result
	}

	// Negative: convert to two's complement
	// Two's complement = 2^(size*8) + val
	modulus := new(big.Int).Lsh(big.NewInt(1), uint(size*8))
	twosComp := new(big.Int).Add(modulus, val)
	bytes := twosComp.Bytes()

	if len(bytes) > size {
		return bytes[len(bytes)-size:]
	}
	result := make([]byte, size)
	copy(result[size-len(bytes):], bytes)
	return result
}

// Helper to pad big.Int bytes to exact size (big-endian, for unsigned integers)
func padBigIntBytesUnsigned(val *big.Int, size int) []byte {
	bytes := val.Bytes()
	if len(bytes) > size {
		return bytes[len(bytes)-size:]
	}
	result := make([]byte, size)
	copy(result[size-len(bytes):], bytes)
	return result
}

// BinaryBuiltins contains the binary module functions for encoding/decoding
var BinaryBuiltins = map[string]*object.Builtin{
	// ============================================================================
	// 8-bit Encoding/Decoding (no endianness needed)
	// ============================================================================

	// encode_i8 encodes a signed 8-bit integer to a byte array.
	// Takes an i8 value. Returns ([byte], Error) tuple.
	"binary.encode_i8": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i8() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i8()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < -128 || intVal > 127 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of i8 range (-128 to 127)", intVal)),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray([]byte{byte(int8(intVal))}),
				object.NIL,
			}}
		},
	},

	// decode_i8 decodes a byte array to a signed 8-bit integer.
	// Takes a 1-byte array. Returns (i8, Error) tuple.
	"binary.decode_i8": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i8() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 1, "binary.decode_i8()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := int8(data[0])
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "i8"},
				object.NIL,
			}}
		},
	},

	// encode_u8 encodes an unsigned 8-bit integer to a byte array.
	// Takes a u8 value. Returns ([byte], Error) tuple.
	"binary.encode_u8": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u8() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u8()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < 0 || intVal > 255 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of u8 range (0 to 255)", intVal)),
				}}
			}
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray([]byte{byte(intVal)}),
				object.NIL,
			}}
		},
	},

	// decode_u8 decodes a byte array to an unsigned 8-bit integer.
	// Takes a 1-byte array. Returns (u8, Error) tuple.
	"binary.decode_u8": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u8() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 1, "binary.decode_u8()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(data[0])), DeclaredType: "u8"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 16-bit Encoding (Little Endian)
	// ============================================================================

	// encode_i16_to_little_endian encodes an i16 to little-endian byte array.
	// Takes an i16 value. Returns ([byte], Error) tuple.
	"binary.encode_i16_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i16_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i16_to_little_endian()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < -32768 || intVal > 32767 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of i16 range", intVal)),
				}}
			}
			buf := make([]byte, 2)
			binary.LittleEndian.PutUint16(buf, uint16(int16(intVal)))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	// decode_i16_from_little_endian decodes a little-endian byte array to i16.
	// Takes a 2-byte array. Returns (i16, Error) tuple.
	"binary.decode_i16_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i16_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 2, "binary.decode_i16_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := int16(binary.LittleEndian.Uint16(data))
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "i16"},
				object.NIL,
			}}
		},
	},

	// encode_u16_to_little_endian encodes a u16 to little-endian byte array.
	// Takes a u16 value. Returns ([byte], Error) tuple.
	"binary.encode_u16_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u16_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u16_to_little_endian()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < 0 || intVal > 65535 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of u16 range", intVal)),
				}}
			}
			buf := make([]byte, 2)
			binary.LittleEndian.PutUint16(buf, uint16(intVal))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	// decode_u16_from_little_endian decodes a little-endian byte array to u16.
	// Takes a 2-byte array. Returns (u16, Error) tuple.
	"binary.decode_u16_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u16_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 2, "binary.decode_u16_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := binary.LittleEndian.Uint16(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "u16"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 16-bit Encoding (Big Endian)
	// ============================================================================

	// encode_i16_to_big_endian encodes an i16 to big-endian byte array.
	// Takes an i16 value. Returns ([byte], Error) tuple.
	"binary.encode_i16_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i16_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i16_to_big_endian()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < -32768 || intVal > 32767 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of i16 range", intVal)),
				}}
			}
			buf := make([]byte, 2)
			binary.BigEndian.PutUint16(buf, uint16(int16(intVal)))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_i16_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i16_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 2, "binary.decode_i16_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := int16(binary.BigEndian.Uint16(data))
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "i16"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u16_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u16_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u16_to_big_endian()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < 0 || intVal > 65535 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of u16 range", intVal)),
				}}
			}
			buf := make([]byte, 2)
			binary.BigEndian.PutUint16(buf, uint16(intVal))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_u16_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u16_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 2, "binary.decode_u16_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := binary.BigEndian.Uint16(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "u16"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 32-bit Encoding (Little Endian)
	// ============================================================================

	// encode_i32_to_little_endian encodes an i32 to little-endian byte array.
	// Takes an i32 value. Returns ([byte], Error) tuple.
	"binary.encode_i32_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i32_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i32_to_little_endian()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < -2147483648 || intVal > 2147483647 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of i32 range", intVal)),
				}}
			}
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(int32(intVal)))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_i32_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i32_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 4, "binary.decode_i32_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := int32(binary.LittleEndian.Uint32(data))
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "i32"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u32_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u32_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u32_to_little_endian()")
			if err != nil {
				return err
			}
			if val.Sign() < 0 || val.Cmp(big.NewInt(4294967295)) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %s out of u32 range", val.String())),
				}}
			}
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, uint32(val.Int64()))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_u32_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u32_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 4, "binary.decode_u32_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := binary.LittleEndian.Uint32(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "u32"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 32-bit Encoding (Big Endian)
	// ============================================================================

	// encode_i32_to_big_endian encodes an i32 to big-endian byte array.
	// Takes an i32 value. Returns ([byte], Error) tuple.
	"binary.encode_i32_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i32_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i32_to_big_endian()")
			if err != nil {
				return err
			}
			intVal := val.Int64()
			if intVal < -2147483648 || intVal > 2147483647 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %d out of i32 range", intVal)),
				}}
			}
			buf := make([]byte, 4)
			binary.BigEndian.PutUint32(buf, uint32(int32(intVal)))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_i32_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i32_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 4, "binary.decode_i32_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := int32(binary.BigEndian.Uint32(data))
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "i32"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u32_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u32_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u32_to_big_endian()")
			if err != nil {
				return err
			}
			if val.Sign() < 0 || val.Cmp(big.NewInt(4294967295)) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %s out of u32 range", val.String())),
				}}
			}
			buf := make([]byte, 4)
			binary.BigEndian.PutUint32(buf, uint32(val.Int64()))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_u32_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u32_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 4, "binary.decode_u32_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := binary.BigEndian.Uint32(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(int64(val)), DeclaredType: "u32"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 64-bit Encoding (Little Endian)
	// ============================================================================

	// encode_i64_to_little_endian encodes an i64 to little-endian byte array.
	// Takes an i64 value. Returns ([byte], Error) tuple.
	"binary.encode_i64_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i64_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i64_to_little_endian()")
			if err != nil {
				return err
			}
			minI64 := new(big.Int).SetInt64(-9223372036854775808)
			maxI64 := new(big.Int).SetInt64(9223372036854775807)
			if val.Cmp(minI64) < 0 || val.Cmp(maxI64) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %s out of i64 range", val.String())),
				}}
			}
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, uint64(val.Int64()))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_i64_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i64_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 8, "binary.decode_i64_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := int64(binary.LittleEndian.Uint64(data))
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(val), DeclaredType: "i64"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u64_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u64_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u64_to_little_endian()")
			if err != nil {
				return err
			}
			maxU64 := new(big.Int).SetUint64(18446744073709551615)
			if val.Sign() < 0 || val.Cmp(maxU64) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %s out of u64 range", val.String())),
				}}
			}
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, val.Uint64())
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_u64_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u64_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 8, "binary.decode_u64_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := binary.LittleEndian.Uint64(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: new(big.Int).SetUint64(val), DeclaredType: "u64"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 64-bit Encoding (Big Endian)
	// ============================================================================

	// encode_i64_to_big_endian encodes an i64 to big-endian byte array.
	// Takes an i64 value. Returns ([byte], Error) tuple.
	"binary.encode_i64_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i64_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i64_to_big_endian()")
			if err != nil {
				return err
			}
			minI64 := new(big.Int).SetInt64(-9223372036854775808)
			maxI64 := new(big.Int).SetInt64(9223372036854775807)
			if val.Cmp(minI64) < 0 || val.Cmp(maxI64) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %s out of i64 range", val.String())),
				}}
			}
			buf := make([]byte, 8)
			binary.BigEndian.PutUint64(buf, uint64(val.Int64()))
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_i64_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i64_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 8, "binary.decode_i64_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := int64(binary.BigEndian.Uint64(data))
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: big.NewInt(val), DeclaredType: "i64"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u64_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u64_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u64_to_big_endian()")
			if err != nil {
				return err
			}
			maxU64 := new(big.Int).SetUint64(18446744073709551615)
			if val.Sign() < 0 || val.Cmp(maxU64) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", fmt.Sprintf("value %s out of u64 range", val.String())),
				}}
			}
			buf := make([]byte, 8)
			binary.BigEndian.PutUint64(buf, val.Uint64())
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_u64_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u64_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 8, "binary.decode_u64_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := binary.BigEndian.Uint64(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: new(big.Int).SetUint64(val), DeclaredType: "u64"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 128-bit Encoding (Little Endian)
	// ============================================================================

	// encode_i128_to_little_endian encodes an i128 to little-endian byte array.
	// Takes an i128 value. Returns ([byte], Error) tuple.
	"binary.encode_i128_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i128_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i128_to_little_endian()")
			if err != nil {
				return err
			}
			minI128 := new(big.Int).Lsh(big.NewInt(-1), 127)
			maxI128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))
			if val.Cmp(minI128) < 0 || val.Cmp(maxI128) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of i128 range"),
				}}
			}
			buf := padBigIntBytesSigned(val, 16)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(reverseBytes(buf)),
				object.NIL,
			}}
		},
	},

	"binary.decode_i128_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i128_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 16, "binary.decode_i128_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			// Reverse for big-endian interpretation
			beData := reverseBytes(data)
			val := new(big.Int).SetBytes(beData)
			// Check if negative (high bit set)
			if beData[0]&0x80 != 0 {
				// Two's complement: subtract 2^128
				twoTo128 := new(big.Int).Lsh(big.NewInt(1), 128)
				val.Sub(val, twoTo128)
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "i128"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u128_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u128_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u128_to_little_endian()")
			if err != nil {
				return err
			}
			maxU128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
			if val.Sign() < 0 || val.Cmp(maxU128) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of u128 range"),
				}}
			}
			buf := padBigIntBytesUnsigned(val, 16)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(reverseBytes(buf)),
				object.NIL,
			}}
		},
	},

	"binary.decode_u128_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u128_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 16, "binary.decode_u128_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			beData := reverseBytes(data)
			val := new(big.Int).SetBytes(beData)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "u128"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 128-bit Encoding (Big Endian)
	// ============================================================================

	// encode_i128_to_big_endian encodes an i128 to big-endian byte array.
	// Takes an i128 value. Returns ([byte], Error) tuple.
	"binary.encode_i128_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i128_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i128_to_big_endian()")
			if err != nil {
				return err
			}
			minI128 := new(big.Int).Lsh(big.NewInt(-1), 127)
			maxI128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))
			if val.Cmp(minI128) < 0 || val.Cmp(maxI128) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of i128 range"),
				}}
			}
			buf := padBigIntBytesSigned(val, 16)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_i128_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i128_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 16, "binary.decode_i128_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := new(big.Int).SetBytes(data)
			// Check if negative (high bit set)
			if data[0]&0x80 != 0 {
				twoTo128 := new(big.Int).Lsh(big.NewInt(1), 128)
				val.Sub(val, twoTo128)
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "i128"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u128_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u128_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u128_to_big_endian()")
			if err != nil {
				return err
			}
			maxU128 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 128), big.NewInt(1))
			if val.Sign() < 0 || val.Cmp(maxU128) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of u128 range"),
				}}
			}
			buf := padBigIntBytesUnsigned(val, 16)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_u128_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u128_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 16, "binary.decode_u128_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := new(big.Int).SetBytes(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "u128"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 256-bit Encoding (Little Endian)
	// ============================================================================

	// encode_i256_to_little_endian encodes an i256 to little-endian byte array.
	// Takes an i256 value. Returns ([byte], Error) tuple.
	"binary.encode_i256_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i256_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i256_to_little_endian()")
			if err != nil {
				return err
			}
			minI256 := new(big.Int).Lsh(big.NewInt(-1), 255)
			maxI256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
			if val.Cmp(minI256) < 0 || val.Cmp(maxI256) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of i256 range"),
				}}
			}
			buf := padBigIntBytesSigned(val, 32)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(reverseBytes(buf)),
				object.NIL,
			}}
		},
	},

	"binary.decode_i256_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i256_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 32, "binary.decode_i256_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			beData := reverseBytes(data)
			val := new(big.Int).SetBytes(beData)
			if beData[0]&0x80 != 0 {
				twoTo256 := new(big.Int).Lsh(big.NewInt(1), 256)
				val.Sub(val, twoTo256)
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "i256"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u256_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u256_to_little_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u256_to_little_endian()")
			if err != nil {
				return err
			}
			maxU256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
			if val.Sign() < 0 || val.Cmp(maxU256) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of u256 range"),
				}}
			}
			buf := padBigIntBytesUnsigned(val, 32)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(reverseBytes(buf)),
				object.NIL,
			}}
		},
	},

	"binary.decode_u256_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u256_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 32, "binary.decode_u256_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			beData := reverseBytes(data)
			val := new(big.Int).SetBytes(beData)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "u256"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// 256-bit Encoding (Big Endian)
	// ============================================================================

	// encode_i256_to_big_endian encodes an i256 to big-endian byte array.
	// Takes an i256 value. Returns ([byte], Error) tuple.
	"binary.encode_i256_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_i256_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_i256_to_big_endian()")
			if err != nil {
				return err
			}
			minI256 := new(big.Int).Lsh(big.NewInt(-1), 255)
			maxI256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(1))
			if val.Cmp(minI256) < 0 || val.Cmp(maxI256) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of i256 range"),
				}}
			}
			buf := padBigIntBytesSigned(val, 32)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_i256_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_i256_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 32, "binary.decode_i256_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := new(big.Int).SetBytes(data)
			if data[0]&0x80 != 0 {
				twoTo256 := new(big.Int).Lsh(big.NewInt(1), 256)
				val.Sub(val, twoTo256)
			}
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "i256"},
				object.NIL,
			}}
		},
	},

	"binary.encode_u256_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_u256_to_big_endian() takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], "binary.encode_u256_to_big_endian()")
			if err != nil {
				return err
			}
			maxU256 := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
			if val.Sign() < 0 || val.Cmp(maxU256) > 0 {
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", "value out of u256 range"),
				}}
			}
			buf := padBigIntBytesUnsigned(val, 32)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_u256_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_u256_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 32, "binary.decode_u256_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			val := new(big.Int).SetBytes(data)
			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: val, DeclaredType: "u256"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Float Encoding (Little Endian)
	// ============================================================================

	// encode_f32_to_little_endian encodes an f32 to little-endian byte array.
	// Takes an f32 value. Returns ([byte], Error) tuple.
	"binary.encode_f32_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_f32_to_little_endian() takes exactly 1 argument"}
			}
			var floatVal float64
			switch v := args[0].(type) {
			case *object.Float:
				floatVal = v.Value
			case *object.Integer:
				f, _ := new(big.Float).SetInt(v.Value).Float64()
				floatVal = f
			default:
				return &object.Error{Code: "E7004", Message: "binary.encode_f32_to_little_endian() requires a float or integer argument"}
			}
			bits := math.Float32bits(float32(floatVal))
			buf := make([]byte, 4)
			binary.LittleEndian.PutUint32(buf, bits)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_f32_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_f32_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 4, "binary.decode_f32_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			bits := binary.LittleEndian.Uint32(data)
			val := math.Float32frombits(bits)
			return &object.ReturnValue{Values: []object.Object{
				&object.Float{Value: float64(val), DeclaredType: "f32"},
				object.NIL,
			}}
		},
	},

	"binary.encode_f64_to_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_f64_to_little_endian() takes exactly 1 argument"}
			}
			var floatVal float64
			switch v := args[0].(type) {
			case *object.Float:
				floatVal = v.Value
			case *object.Integer:
				f, _ := new(big.Float).SetInt(v.Value).Float64()
				floatVal = f
			default:
				return &object.Error{Code: "E7004", Message: "binary.encode_f64_to_little_endian() requires a float or integer argument"}
			}
			bits := math.Float64bits(floatVal)
			buf := make([]byte, 8)
			binary.LittleEndian.PutUint64(buf, bits)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_f64_from_little_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_f64_from_little_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 8, "binary.decode_f64_from_little_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			bits := binary.LittleEndian.Uint64(data)
			val := math.Float64frombits(bits)
			return &object.ReturnValue{Values: []object.Object{
				&object.Float{Value: val, DeclaredType: "f64"},
				object.NIL,
			}}
		},
	},

	// ============================================================================
	// Float Encoding (Big Endian)
	// ============================================================================

	// encode_f32_to_big_endian encodes an f32 to big-endian byte array.
	// Takes an f32 value. Returns ([byte], Error) tuple.
	"binary.encode_f32_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_f32_to_big_endian() takes exactly 1 argument"}
			}
			var floatVal float64
			switch v := args[0].(type) {
			case *object.Float:
				floatVal = v.Value
			case *object.Integer:
				f, _ := new(big.Float).SetInt(v.Value).Float64()
				floatVal = f
			default:
				return &object.Error{Code: "E7004", Message: "binary.encode_f32_to_big_endian() requires a float or integer argument"}
			}
			bits := math.Float32bits(float32(floatVal))
			buf := make([]byte, 4)
			binary.BigEndian.PutUint32(buf, bits)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_f32_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_f32_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 4, "binary.decode_f32_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			bits := binary.BigEndian.Uint32(data)
			val := math.Float32frombits(bits)
			return &object.ReturnValue{Values: []object.Object{
				&object.Float{Value: float64(val), DeclaredType: "f32"},
				object.NIL,
			}}
		},
	},

	"binary.encode_f64_to_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.encode_f64_to_big_endian() takes exactly 1 argument"}
			}
			var floatVal float64
			switch v := args[0].(type) {
			case *object.Float:
				floatVal = v.Value
			case *object.Integer:
				f, _ := new(big.Float).SetInt(v.Value).Float64()
				floatVal = f
			default:
				return &object.Error{Code: "E7004", Message: "binary.encode_f64_to_big_endian() requires a float or integer argument"}
			}
			bits := math.Float64bits(floatVal)
			buf := make([]byte, 8)
			binary.BigEndian.PutUint64(buf, bits)
			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	},

	"binary.decode_f64_from_big_endian": {
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: "binary.decode_f64_from_big_endian() takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], 8, "binary.decode_f64_from_big_endian()")
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}
			bits := binary.BigEndian.Uint64(data)
			val := math.Float64frombits(bits)
			return &object.ReturnValue{Values: []object.Object{
				&object.Float{Value: val, DeclaredType: "f64"},
				object.NIL,
			}}
		},
	},
}

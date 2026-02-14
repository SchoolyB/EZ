package stdlib

// Copyright (c) 2025-Present Marshall A Burns
// Licensed under the MIT License. See LICENSE for details.

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/big"

	"github.com/marshallburns/ez/pkg/errors"
	"github.com/marshallburns/ez/pkg/object"
)

// Helper to convert EZ byte array to Go []byte with length validation
func binaryBytesToSlice(arg object.Object, expected int, funcName string) ([]byte, *object.Struct) {
	arr, ok := arg.(*object.Array)
	if !ok {
		return nil, CreateStdlibError("E7002", fmt.Sprintf("%s requires a byte array argument", errors.Ident(funcName)))
	}
	if len(arr.Elements) != expected {
		return nil, CreateStdlibError("E7010", fmt.Sprintf("%s requires exactly %d bytes, got %d", errors.Ident(funcName), expected, len(arr.Elements)))
	}

	data := make([]byte, expected)
	for i, elem := range arr.Elements {
		switch e := elem.(type) {
		case *object.Byte:
			data[i] = e.Value
		case *object.Integer:
			val := e.Value.Int64()
			if val < 0 || val > 255 {
				return nil, CreateStdlibError("E3022", fmt.Sprintf("%s byte at index %d out of range (0-255)", errors.Ident(funcName), i))
			}
			data[i] = byte(val)
		default:
			return nil, CreateStdlibError("E7002", fmt.Sprintf("%s requires a byte array", errors.Ident(funcName)))
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
			Message: fmt.Sprintf("%s requires an integer argument", errors.Ident(funcName)),
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

// computeIntBounds returns the min and max values for an integer of the given bit width
func computeIntBounds(bitWidth int, signed bool) (min, max *big.Int) {
	if signed {
		min = new(big.Int).Lsh(big.NewInt(-1), uint(bitWidth-1))
		max = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bitWidth-1)), big.NewInt(1))
	} else {
		min = big.NewInt(0)
		max = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), uint(bitWidth)), big.NewInt(1))
	}
	return
}

// createIntEncoder creates a builtin for encoding an integer to bytes
func createIntEncoder(bitWidth int, signed bool, endianness binary.ByteOrder, funcName string) *object.Builtin {
	byteSize := bitWidth / 8
	minVal, maxVal := computeIntBounds(bitWidth, signed)
	prefix := "u"
	if signed {
		prefix = "i"
	}
	typeName := fmt.Sprintf("%s%d", prefix, bitWidth)

	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: funcName + " takes exactly 1 argument"}
			}
			val, err := extractEncodingInt(args[0], funcName)
			if err != nil {
				return err
			}

			if val.Cmp(minVal) < 0 || val.Cmp(maxVal) > 0 {
				var errMsg string
				if bitWidth <= 32 {
					errMsg = fmt.Sprintf("value %d out of %s range", val.Int64(), typeName)
				} else if bitWidth <= 64 {
					errMsg = fmt.Sprintf("value %s out of %s range", val.String(), typeName)
				} else {
					errMsg = fmt.Sprintf("value out of %s range", typeName)
				}
				return &object.ReturnValue{Values: []object.Object{
					object.NIL,
					CreateStdlibError("E3022", errMsg),
				}}
			}

			var buf []byte
			switch bitWidth {
			case 16:
				buf = make([]byte, 2)
				if signed {
					endianness.PutUint16(buf, uint16(int16(val.Int64())))
				} else {
					endianness.PutUint16(buf, uint16(val.Int64()))
				}
			case 32:
				buf = make([]byte, 4)
				if signed {
					endianness.PutUint32(buf, uint32(int32(val.Int64())))
				} else {
					endianness.PutUint32(buf, uint32(val.Int64()))
				}
			case 64:
				buf = make([]byte, 8)
				if signed {
					endianness.PutUint64(buf, uint64(val.Int64()))
				} else {
					endianness.PutUint64(buf, val.Uint64())
				}
			default: // 128, 256
				if signed {
					buf = padBigIntBytesSigned(val, byteSize)
				} else {
					buf = padBigIntBytesUnsigned(val, byteSize)
				}
				if endianness == binary.LittleEndian {
					buf = reverseBytes(buf)
				}
			}

			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	}
}

// createIntDecoder creates a builtin for decoding bytes to an integer
func createIntDecoder(bitWidth int, signed bool, endianness binary.ByteOrder, funcName string) *object.Builtin {
	byteSize := bitWidth / 8
	prefix := "u"
	if signed {
		prefix = "i"
	}
	typeName := fmt.Sprintf("%s%d", prefix, bitWidth)

	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: funcName + " takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], byteSize, funcName)
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}

			var result *big.Int
			switch bitWidth {
			case 16:
				raw := endianness.Uint16(data)
				if signed {
					result = big.NewInt(int64(int16(raw)))
				} else {
					result = big.NewInt(int64(raw))
				}
			case 32:
				raw := endianness.Uint32(data)
				if signed {
					result = big.NewInt(int64(int32(raw)))
				} else {
					result = big.NewInt(int64(raw))
				}
			case 64:
				raw := endianness.Uint64(data)
				if signed {
					result = big.NewInt(int64(raw))
				} else {
					result = new(big.Int).SetUint64(raw)
				}
			default: // 128, 256
				beData := data
				if endianness == binary.LittleEndian {
					beData = reverseBytes(data)
				}
				result = new(big.Int).SetBytes(beData)
				if signed && beData[0]&0x80 != 0 {
					modulus := new(big.Int).Lsh(big.NewInt(1), uint(bitWidth))
					result.Sub(result, modulus)
				}
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Integer{Value: result, DeclaredType: typeName},
				object.NIL,
			}}
		},
	}
}

// createFloatEncoder creates a builtin for encoding a float to bytes
func createFloatEncoder(bitWidth int, endianness binary.ByteOrder, funcName string) *object.Builtin {
	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: funcName + " takes exactly 1 argument"}
			}
			var floatVal float64
			switch v := args[0].(type) {
			case *object.Float:
				floatVal = v.Value
			case *object.Integer:
				f, _ := new(big.Float).SetInt(v.Value).Float64()
				floatVal = f
			default:
				return &object.Error{Code: "E7004", Message: funcName + " requires a float or integer argument"}
			}

			var buf []byte
			if bitWidth == 32 {
				bits := math.Float32bits(float32(floatVal))
				buf = make([]byte, 4)
				endianness.PutUint32(buf, bits)
			} else {
				bits := math.Float64bits(floatVal)
				buf = make([]byte, 8)
				endianness.PutUint64(buf, bits)
			}

			return &object.ReturnValue{Values: []object.Object{
				sliceToBinaryArray(buf),
				object.NIL,
			}}
		},
	}
}

// createFloatDecoder creates a builtin for decoding bytes to a float
func createFloatDecoder(bitWidth int, endianness binary.ByteOrder, funcName string) *object.Builtin {
	byteSize := bitWidth / 8
	typeName := fmt.Sprintf("f%d", bitWidth)

	return &object.Builtin{
		Fn: func(args ...object.Object) object.Object {
			if len(args) != 1 {
				return &object.Error{Code: "E7001", Message: funcName + " takes exactly 1 argument"}
			}
			data, errStruct := binaryBytesToSlice(args[0], byteSize, funcName)
			if errStruct != nil {
				return &object.ReturnValue{Values: []object.Object{object.NIL, errStruct}}
			}

			var floatResult float64
			if bitWidth == 32 {
				bits := endianness.Uint32(data)
				floatResult = float64(math.Float32frombits(bits))
			} else {
				bits := endianness.Uint64(data)
				floatResult = math.Float64frombits(bits)
			}

			return &object.ReturnValue{Values: []object.Object{
				&object.Float{Value: floatResult, DeclaredType: typeName},
				object.NIL,
			}}
		},
	}
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
	// 16-bit through 256-bit Integer Encoding/Decoding
	// ============================================================================

	"binary.encode_i16_to_little_endian":   createIntEncoder(16, true, binary.LittleEndian, "binary.encode_i16_to_little_endian()"),
	"binary.decode_i16_from_little_endian": createIntDecoder(16, true, binary.LittleEndian, "binary.decode_i16_from_little_endian()"),
	"binary.encode_u16_to_little_endian":   createIntEncoder(16, false, binary.LittleEndian, "binary.encode_u16_to_little_endian()"),
	"binary.decode_u16_from_little_endian": createIntDecoder(16, false, binary.LittleEndian, "binary.decode_u16_from_little_endian()"),
	"binary.encode_i16_to_big_endian":      createIntEncoder(16, true, binary.BigEndian, "binary.encode_i16_to_big_endian()"),
	"binary.decode_i16_from_big_endian":    createIntDecoder(16, true, binary.BigEndian, "binary.decode_i16_from_big_endian()"),
	"binary.encode_u16_to_big_endian":      createIntEncoder(16, false, binary.BigEndian, "binary.encode_u16_to_big_endian()"),
	"binary.decode_u16_from_big_endian":    createIntDecoder(16, false, binary.BigEndian, "binary.decode_u16_from_big_endian()"),

	"binary.encode_i32_to_little_endian":   createIntEncoder(32, true, binary.LittleEndian, "binary.encode_i32_to_little_endian()"),
	"binary.decode_i32_from_little_endian": createIntDecoder(32, true, binary.LittleEndian, "binary.decode_i32_from_little_endian()"),
	"binary.encode_u32_to_little_endian":   createIntEncoder(32, false, binary.LittleEndian, "binary.encode_u32_to_little_endian()"),
	"binary.decode_u32_from_little_endian": createIntDecoder(32, false, binary.LittleEndian, "binary.decode_u32_from_little_endian()"),
	"binary.encode_i32_to_big_endian":      createIntEncoder(32, true, binary.BigEndian, "binary.encode_i32_to_big_endian()"),
	"binary.decode_i32_from_big_endian":    createIntDecoder(32, true, binary.BigEndian, "binary.decode_i32_from_big_endian()"),
	"binary.encode_u32_to_big_endian":      createIntEncoder(32, false, binary.BigEndian, "binary.encode_u32_to_big_endian()"),
	"binary.decode_u32_from_big_endian":    createIntDecoder(32, false, binary.BigEndian, "binary.decode_u32_from_big_endian()"),

	"binary.encode_i64_to_little_endian":   createIntEncoder(64, true, binary.LittleEndian, "binary.encode_i64_to_little_endian()"),
	"binary.decode_i64_from_little_endian": createIntDecoder(64, true, binary.LittleEndian, "binary.decode_i64_from_little_endian()"),
	"binary.encode_u64_to_little_endian":   createIntEncoder(64, false, binary.LittleEndian, "binary.encode_u64_to_little_endian()"),
	"binary.decode_u64_from_little_endian": createIntDecoder(64, false, binary.LittleEndian, "binary.decode_u64_from_little_endian()"),
	"binary.encode_i64_to_big_endian":      createIntEncoder(64, true, binary.BigEndian, "binary.encode_i64_to_big_endian()"),
	"binary.decode_i64_from_big_endian":    createIntDecoder(64, true, binary.BigEndian, "binary.decode_i64_from_big_endian()"),
	"binary.encode_u64_to_big_endian":      createIntEncoder(64, false, binary.BigEndian, "binary.encode_u64_to_big_endian()"),
	"binary.decode_u64_from_big_endian":    createIntDecoder(64, false, binary.BigEndian, "binary.decode_u64_from_big_endian()"),

	"binary.encode_i128_to_little_endian":   createIntEncoder(128, true, binary.LittleEndian, "binary.encode_i128_to_little_endian()"),
	"binary.decode_i128_from_little_endian": createIntDecoder(128, true, binary.LittleEndian, "binary.decode_i128_from_little_endian()"),
	"binary.encode_u128_to_little_endian":   createIntEncoder(128, false, binary.LittleEndian, "binary.encode_u128_to_little_endian()"),
	"binary.decode_u128_from_little_endian": createIntDecoder(128, false, binary.LittleEndian, "binary.decode_u128_from_little_endian()"),
	"binary.encode_i128_to_big_endian":      createIntEncoder(128, true, binary.BigEndian, "binary.encode_i128_to_big_endian()"),
	"binary.decode_i128_from_big_endian":    createIntDecoder(128, true, binary.BigEndian, "binary.decode_i128_from_big_endian()"),
	"binary.encode_u128_to_big_endian":      createIntEncoder(128, false, binary.BigEndian, "binary.encode_u128_to_big_endian()"),
	"binary.decode_u128_from_big_endian":    createIntDecoder(128, false, binary.BigEndian, "binary.decode_u128_from_big_endian()"),

	"binary.encode_i256_to_little_endian":   createIntEncoder(256, true, binary.LittleEndian, "binary.encode_i256_to_little_endian()"),
	"binary.decode_i256_from_little_endian": createIntDecoder(256, true, binary.LittleEndian, "binary.decode_i256_from_little_endian()"),
	"binary.encode_u256_to_little_endian":   createIntEncoder(256, false, binary.LittleEndian, "binary.encode_u256_to_little_endian()"),
	"binary.decode_u256_from_little_endian": createIntDecoder(256, false, binary.LittleEndian, "binary.decode_u256_from_little_endian()"),
	"binary.encode_i256_to_big_endian":      createIntEncoder(256, true, binary.BigEndian, "binary.encode_i256_to_big_endian()"),
	"binary.decode_i256_from_big_endian":    createIntDecoder(256, true, binary.BigEndian, "binary.decode_i256_from_big_endian()"),
	"binary.encode_u256_to_big_endian":      createIntEncoder(256, false, binary.BigEndian, "binary.encode_u256_to_big_endian()"),
	"binary.decode_u256_from_big_endian":    createIntDecoder(256, false, binary.BigEndian, "binary.decode_u256_from_big_endian()"),

	// ============================================================================
	// Float Encoding/Decoding
	// ============================================================================

	"binary.encode_f32_to_little_endian":   createFloatEncoder(32, binary.LittleEndian, "binary.encode_f32_to_little_endian()"),
	"binary.decode_f32_from_little_endian": createFloatDecoder(32, binary.LittleEndian, "binary.decode_f32_from_little_endian()"),
	"binary.encode_f64_to_little_endian":   createFloatEncoder(64, binary.LittleEndian, "binary.encode_f64_to_little_endian()"),
	"binary.decode_f64_from_little_endian": createFloatDecoder(64, binary.LittleEndian, "binary.decode_f64_from_little_endian()"),
	"binary.encode_f32_to_big_endian":      createFloatEncoder(32, binary.BigEndian, "binary.encode_f32_to_big_endian()"),
	"binary.decode_f32_from_big_endian":    createFloatDecoder(32, binary.BigEndian, "binary.decode_f32_from_big_endian()"),
	"binary.encode_f64_to_big_endian":      createFloatEncoder(64, binary.BigEndian, "binary.encode_f64_to_big_endian()"),
	"binary.decode_f64_from_big_endian":    createFloatDecoder(64, binary.BigEndian, "binary.decode_f64_from_big_endian()"),
}

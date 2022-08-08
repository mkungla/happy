// Copyright 2022 The Happy Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vars

import (
	"fmt"
	"strings"
)

// Value describes the value.
type Value struct {
	vtype Type
	raw   any
	str   string
}

// Type of value.
func (v Value) Type() Type {
	return v.vtype
}

// Bool returns boolean representation of the var Value.
func (v Value) Bool() bool {
	if v.vtype == TypeBool {
		if v, ok := v.raw.(bool); ok {
			return v
		}
	}
	val, _, _ := parseBool(v.str)
	return val
}

// Float32 returns Float32 representation of the Value.
func (v Value) Float32() float32 {
	if v.vtype == TypeFloat32 {
		if v, ok := v.raw.(float32); ok {
			return v
		}
	}
	val, _, _ := parseFloat(v.str, 32)
	return float32(val)
}

// Float64 returns float64 representation of Value.
func (v Value) Float64() float64 {
	if v.vtype == TypeFloat64 {
		if v, ok := v.raw.(float64); ok {
			return v
		}
	}
	val, _, _ := parseFloat(v.str, 64)
	return val
}

// Complex64 returns complex64 representation of the Value.
func (v Value) Complex64() complex64 {
	if v.vtype == TypeComplex64 {
		if v, ok := v.raw.(complex64); ok {
			return v
		}
	}
	val, _, _ := parseComplex64(v.str)
	return val
}

// Complex128 returns complex128 representation of the Value.
func (v Value) Complex128() complex128 {
	if v.vtype == TypeComplex128 {
		if v, ok := v.raw.(complex128); ok {
			return v
		}
	}
	val, _, _ := parseComplex128(v.str)
	return val
}

// Int returns int representation of the Value.
func (v Value) Int() int {
	if v.vtype == TypeInt {
		if v, ok := v.raw.(int); ok {
			return v
		}
	}
	val, _, _ := parseInt(v.str, 10, 0)
	return int(val)
}

// Int8 returns int8 representation of the Value.
func (v Value) Int8() int8 {
	if v.vtype == TypeInt8 {
		if v, ok := v.raw.(int8); ok {
			return v
		}
	}
	val, _, _ := parseInt(v.str, 10, 8)
	return int8(val)
}

// Int16 returns int16 representation of the Value.
func (v Value) Int16() int16 {
	val, _, _ := parseInt(v.str, 10, 16)
	return int16(val)
}

// Int32 returns int32 representation of the Value.
func (v Value) Int32() int32 {
	val, _, _ := parseInt(v.str, 10, 32)
	return int32(val)
}

// Int64 returns int64 representation of the Value.
func (v Value) Int64() int64 {
	val, _, _ := parseInt(v.str, 10, 64)
	return val
}

// Uint returns uint representation of the Value.
func (v Value) Uint() uint {
	if v.vtype == TypeUint {
		if v, ok := v.raw.(uint); ok {
			return v
		}
	}
	val, _, _ := parseUint(v.str, 10, 0)
	return uint(val)
}

// Uint8 returns uint8 representation of the Value.
func (v Value) Uint8() uint8 {
	if v.vtype == TypeUint8 {
		if v, ok := v.raw.(uint8); ok {
			return v
		}
	}
	val, _, _ := parseUint(v.str, 10, 8)
	return uint8(val)
}

// Uint16 returns uint16 representation of the Value.
func (v Value) Uint16() uint16 {
	if v.vtype == TypeUint16 {
		if v, ok := v.raw.(uint16); ok {
			return v
		}
	}
	val, _, _ := parseUint(v.str, 10, 16)
	return uint16(val)
}

// Uint32 returns uint32 representation of the Value.
func (v Value) Uint32() uint32 {
	if v.vtype == TypeUint32 {
		if v, ok := v.raw.(uint32); ok {
			return v
		}
	}
	val, _, _ := parseUint(v.str, 10, 32)
	return uint32(val)
}

// Uint64 returns uint64 representation of the Value.
func (v Value) Uint64() uint64 {
	if v.vtype == TypeUint64 {
		if v, ok := v.raw.(uint64); ok {
			return v
		}
	}
	val, _, _ := parseUint(v.str, 10, 64)
	return val
}

// Uintptr returns uintptr representation of the Value.
func (v Value) Uintptr() uintptr {
	if v.vtype == TypeUintptr {
		if v, ok := v.raw.(uintptr); ok {
			return v
		}
	}
	val, _, _ := parseUint(v.str, 10, 64)
	return uintptr(val)
}

// String returns string representation of the Value.
func (v Value) String() string {
	if v.vtype == TypeDuration {
		return fmt.Sprint(v.raw)
	}
	if len(v.str) == 0 && v.raw != nil {
		return fmt.Sprint(v.raw)
	}

	return v.str
}

// Bytes returns []bytes representation of the Value.
func (v Value) Bytes() []byte {
	if v.vtype == TypeBytes {
		if v, ok := v.raw.([]byte); ok {
			return v
		}
	}
	return []byte(v.str)
}

// Runes returns []rune representation of the var value.
func (v Value) Runes() []rune {
	return []rune(v.str)
}

// Len returns the length of the string representation of the Value.
func (v Value) Len() int {
	return len(v.str)
}

// Empty returns true if this Value is empty.
func (v Value) Empty() bool {
	return v.Len() == 0 || v.raw == nil
}

// Fields calls strings.Fields on Value string.
func (v Value) Fields() []string {
	return strings.Fields(v.str)
}

func (v Value) Raw() any {
	return v.raw
}
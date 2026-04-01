package typechecker

import "fmt"

// Type represents a Xuesos++ type.
type Type interface {
	TypeName() string
	Equals(other Type) bool
}

// PrimitiveType represents built-in primitive types.
type PrimitiveType struct {
	Name string
}

func (t *PrimitiveType) TypeName() string         { return t.Name }
func (t *PrimitiveType) Equals(other Type) bool {
	if o, ok := other.(*PrimitiveType); ok {
		return t.Name == o.Name
	}
	return false
}

// Singleton primitive types
var (
	TypeInt     = &PrimitiveType{Name: "int"}
	TypeInt8    = &PrimitiveType{Name: "int8"}
	TypeInt16   = &PrimitiveType{Name: "int16"}
	TypeInt32   = &PrimitiveType{Name: "int32"}
	TypeInt64   = &PrimitiveType{Name: "int64"}
	TypeUint    = &PrimitiveType{Name: "uint"}
	TypeFloat   = &PrimitiveType{Name: "float"}
	TypeFloat32 = &PrimitiveType{Name: "float32"}
	TypeBool    = &PrimitiveType{Name: "bool"}
	TypeStr     = &PrimitiveType{Name: "str"}
	TypeChar    = &PrimitiveType{Name: "char"}
	TypeByte    = &PrimitiveType{Name: "byte"}
	TypeVoid    = &PrimitiveType{Name: "void"}
	TypeNull    = &PrimitiveType{Name: "null"}
)

// ArrayType represents []T.
type ArrayType struct {
	ElementType Type
}

func (t *ArrayType) TypeName() string { return "[]" + t.ElementType.TypeName() }
func (t *ArrayType) Equals(other Type) bool {
	if o, ok := other.(*ArrayType); ok {
		return t.ElementType.Equals(o.ElementType)
	}
	return false
}

// NullableType represents ?T.
type NullableType struct {
	Inner Type
}

func (t *NullableType) TypeName() string { return "?" + t.Inner.TypeName() }
func (t *NullableType) Equals(other Type) bool {
	if o, ok := other.(*NullableType); ok {
		return t.Inner.Equals(o.Inner)
	}
	return false
}

// FuncType represents a function signature.
type FuncType struct {
	ParamTypes []Type
	ReturnType Type
}

func (t *FuncType) TypeName() string {
	params := ""
	for i, p := range t.ParamTypes {
		if i > 0 {
			params += ", "
		}
		params += p.TypeName()
	}
	if t.ReturnType == nil || t.ReturnType.Equals(TypeVoid) {
		return fmt.Sprintf("xuen(%s)", params)
	}
	return fmt.Sprintf("xuen(%s) %s", params, t.ReturnType.TypeName())
}
func (t *FuncType) Equals(other Type) bool {
	o, ok := other.(*FuncType)
	if !ok || len(t.ParamTypes) != len(o.ParamTypes) {
		return false
	}
	for i := range t.ParamTypes {
		if !t.ParamTypes[i].Equals(o.ParamTypes[i]) {
			return false
		}
	}
	if t.ReturnType == nil && o.ReturnType == nil {
		return true
	}
	if t.ReturnType == nil || o.ReturnType == nil {
		return false
	}
	return t.ReturnType.Equals(o.ReturnType)
}

// StructType represents a struct type.
type StructType struct {
	Name   string
	Fields map[string]Type
}

func (t *StructType) TypeName() string      { return t.Name }
func (t *StructType) Equals(other Type) bool {
	if o, ok := other.(*StructType); ok {
		return t.Name == o.Name
	}
	return false
}

// EnumType represents an enum type.
type EnumType struct {
	Name     string
	Variants []string
}

func (t *EnumType) TypeName() string      { return t.Name }
func (t *EnumType) Equals(other Type) bool {
	if o, ok := other.(*EnumType); ok {
		return t.Name == o.Name
	}
	return false
}

// RangeType represents a range (start..end).
type RangeType struct{}

func (t *RangeType) TypeName() string      { return "range" }
func (t *RangeType) Equals(other Type) bool {
	_, ok := other.(*RangeType)
	return ok
}

// ResolveTypeName converts a type name string to a Type.
func ResolveTypeName(name string, structs map[string]*StructType, enums map[string]*EnumType) Type {
	// Handle nullable
	if len(name) > 0 && name[0] == '?' {
		inner := ResolveTypeName(name[1:], structs, enums)
		if inner == nil {
			return nil
		}
		return &NullableType{Inner: inner}
	}
	// Handle array
	if len(name) > 2 && name[:2] == "[]" {
		elem := ResolveTypeName(name[2:], structs, enums)
		if elem == nil {
			return nil
		}
		return &ArrayType{ElementType: elem}
	}
	// Primitives
	switch name {
	case "int":
		return TypeInt
	case "int8":
		return TypeInt8
	case "int16":
		return TypeInt16
	case "int32":
		return TypeInt32
	case "int64":
		return TypeInt64
	case "uint":
		return TypeUint
	case "float":
		return TypeFloat
	case "float32":
		return TypeFloat32
	case "bool":
		return TypeBool
	case "str":
		return TypeStr
	case "char":
		return TypeChar
	case "byte":
		return TypeByte
	case "void", "":
		return TypeVoid
	}
	// Struct
	if st, ok := structs[name]; ok {
		return st
	}
	// Enum
	if en, ok := enums[name]; ok {
		return en
	}
	return nil
}

// IsNumeric returns true if the type is numeric (int or float variants).
func IsNumeric(t Type) bool {
	p, ok := t.(*PrimitiveType)
	if !ok {
		return false
	}
	switch p.Name {
	case "int", "int8", "int16", "int32", "int64", "uint", "float", "float32", "byte":
		return true
	}
	return false
}

// IsInteger returns true if the type is an integer variant.
func IsInteger(t Type) bool {
	p, ok := t.(*PrimitiveType)
	if !ok {
		return false
	}
	switch p.Name {
	case "int", "int8", "int16", "int32", "int64", "uint", "byte":
		return true
	}
	return false
}

// IsFloat returns true if the type is a float variant.
func IsFloat(t Type) bool {
	p, ok := t.(*PrimitiveType)
	if !ok {
		return false
	}
	return p.Name == "float" || p.Name == "float32"
}

// AssignableTo checks if src type can be assigned to dst type.
func AssignableTo(src, dst Type) bool {
	if src.Equals(dst) {
		return true
	}
	// null assignable to nullable
	if src.Equals(TypeNull) {
		if _, ok := dst.(*NullableType); ok {
			return true
		}
	}
	// T assignable to ?T
	if nullable, ok := dst.(*NullableType); ok {
		return src.Equals(nullable.Inner)
	}
	// int -> float promotion
	if IsInteger(src) && IsFloat(dst) {
		return true
	}
	// void is compatible with anything (untyped/unknown)
	if src.Equals(TypeVoid) || dst.Equals(TypeVoid) {
		return true
	}
	// []void assignable to []T and vice versa
	if srcArr, ok := src.(*ArrayType); ok {
		if dstArr, ok := dst.(*ArrayType); ok {
			if srcArr.ElementType.Equals(TypeVoid) || dstArr.ElementType.Equals(TypeVoid) {
				return true
			}
		}
	}
	return false
}

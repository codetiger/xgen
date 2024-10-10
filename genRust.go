// Copyright 2020 - 2024 The xgen Authors. All rights reserved. Use of this
// source code is governed by a BSD-style license that can be found in the
// LICENSE file.
//
// Package xgen written in pure Go providing a set of functions that allow you
// to parse XSD (XML schema files). This library needs Go version 1.10 or
// later.

package xgen

import (
	"fmt"
	"os"
	"reflect"
	"strings"
)

var (
	rustBuildinType = map[string]bool{
		"i8":          true,
		"i16":         true,
		"i32":         true,
		"i64":         true,
		"i128":        true,
		"isize":       true,
		"u8":          true,
		"u16":         true,
		"u32":         true,
		"u64":         true,
		"u128":        true,
		"usize":       true,
		"f32":         true,
		"f64":         true,
		"Vec<char>":   true,
		"Vec<String>": true,
		"Vec<u8>":     true,
		"bool":        true,
		"char":        true,
		"String":      true,
	}
	rustKeywords = map[string]bool{
		"as":       true,
		"break":    true,
		"const":    true,
		"continue": true,
		"crate":    true,
		"dyn":      true,
		"else":     true,
		"enum":     true,
		"extern":   true,
		"false":    true,
		"fn":       true,
		"for":      true,
		"if":       true,
		"impl":     true,
		"in":       true,
		"let":      true,
		"loop":     true,
		"match":    true,
		"mod":      true,
		"move":     true,
		"mut":      true,
		"pub":      true,
		"ref":      true,
		"return":   true,
		"Self":     true,
		"self":     true,
		"static":   true,
		"struct":   true,
		"super":    true,
		"trait":    true,
		"true":     true,
		"type":     true,
		"unsafe":   true,
		"use":      true,
		"where":    true,
		"while":    true,
		"abstract": true,
		"async":    true,
		"await":    true,
		"become":   true,
		"box":      true,
		"do":       true,
		"final":    true,
		"macro":    true,
		"override": true,
		"priv":     true,
		"try":      true,
		"typeof":   true,
		"unsized":  true,
		"virtual":  true,
		"yield":    true,
	}
)

// GenRust generate Go programming language source code for XML schema
// definition files.
func (gen *CodeGenerator) GenRust() error {
	fieldNameCount = make(map[string]int)
	for _, ele := range gen.ProtoTree {
		if ele == nil {
			continue
		}
		funcName := fmt.Sprintf("Rust%s", reflect.TypeOf(ele).String()[6:])
		callFuncByName(gen, funcName, []reflect.Value{reflect.ValueOf(ele)})
	}
	f, err := os.Create(gen.FileWithExtension(".rs"))
	if err != nil {
		return err
	}
	defer f.Close()
	var extern = "use serde::{Deserialize, Serialize};\n"
	source := []byte(fmt.Sprintf("%s\n\n%s\n%s", copyright, extern, gen.Field))
	f.Write(source)
	return err
}

// genRustFieldName generate struct field name for Rust code.
func genRustFieldName(name string) (fieldName string) {
	for _, str := range strings.Split(name, ":") {
		fieldName += MakeFirstUpperCase(str)
	}
	var tmp string
	for _, str := range strings.Split(fieldName, ".") {
		tmp += MakeFirstUpperCase(str)
	}
	fieldName = tmp
	fieldName = ToSnakeCase(strings.Replace(fieldName, "-", "", -1))
	if _, ok := rustKeywords[fieldName]; ok {
		fieldName += "_attr"
	}
	return
}

// genRustStructName generate struct name for Rust code.
func genRustStructName(name string, unique bool) (structName string) {
	for _, str := range strings.Split(name, ":") {
		structName += MakeFirstUpperCase(str)
	}
	var tmp string
	for _, str := range strings.Split(structName, ".") {
		tmp += MakeFirstUpperCase(str)
	}
	structName = tmp
	structName = strings.NewReplacer("-", "", "_", "").Replace(structName)
	if unique {
		fieldNameCount[structName]++
		if count := fieldNameCount[structName]; count != 1 {
			structName = fmt.Sprintf("%s%d", structName, count)
		}
	}
	return
}

// genRustFieldType generate struct field type for Rust code.
func genRustFieldType(name string) string {
	if _, ok := rustBuildinType[name]; ok {
		return name
	}
	fieldType := genRustStructName(name, false)
	if fieldType != "" {
		return fieldType
	}
	return "char"
}

func escapeRustString(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func genRustFieldCode(name string, fieldType string, plural bool, optional bool, restriction *Restriction) string {
	attributes := ""
	// Only add validation attributes if there are restrictions
	// if restriction != nil && !restriction.IsEmpty() {
	// 	// Handle length constraints
	// 	if restriction.MinLength > 0 {
	// 		lengthValidation := fmt.Sprintf("\t#[validate(min_length = %d)]\n", restriction.MinLength)
	// 		attributes += lengthValidation
	// 	}
	// 	if restriction.MaxLength > 0 {
	// 		lengthValidation := fmt.Sprintf("\t#[validate(max_length = %d)]\n", restriction.MaxLength)
	// 		attributes += lengthValidation
	// 	}

	// 	// Handle pattern constraints
	// 	if restriction.Pattern != nil {
	// 		patternStr := escapeRustString(restriction.Pattern.String())
	// 		patternValidation := fmt.Sprintf("\t#[validate(pattern = \"%s\")]\n", patternStr)
	// 		attributes += patternValidation
	// 	}

	// 	// Handle enumerations
	// 	if len(restriction.Enum) > 0 {
	// 		var quotedEnums []string
	// 		for _, enumValue := range restriction.Enum {
	// 			escapedValue := strings.ReplaceAll(enumValue, "\"", "\\\"")
	// 			quotedEnums = append(quotedEnums, fmt.Sprintf("\"%s\"", escapedValue))
	// 		}
	// 		enumValues := strings.Join(quotedEnums, ", ")
	// 		enumValidation := fmt.Sprintf("\t#[validate(enumerate = [%s])]\n", enumValues)
	// 		attributes += enumValidation
	// 	}
	// } else if !isRustBuiltInType(fieldType) {
	// 	attributes += "\t#[validate]\n"
	// }

	fields := genRustFieldType(fieldType)
	if plural {
		fields = "Vec<" + fields + ">"
	}
	if optional {
		fields = "Option<" + fields + ">"
	}

	attributes += fmt.Sprintf("\t#[serde(rename = \"%s\")]\n\tpub %s: %s,\n", genRustFieldRename(name), genRustFieldName(name), fields)
	return attributes
}

func genRustStructCode(name string, doc string, fieldContent string) string {
	content := fmt.Sprintf("\n%s#[derive(Debug, Default, PartialEq, Clone, Serialize, Deserialize)]\npub struct %s {\n%s}\n", genFieldComment(name, doc, "//"), name, fieldContent)
	return content
}

// RustSimpleType generates code for simple type XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustSimpleType(v *SimpleType) {
	if v.List {
		if _, ok := gen.StructAST[v.Name]; !ok {
			fieldType := getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)
			content := genRustFieldCode(v.Name, fieldType, true, false, &v.Restriction)
			gen.StructAST[v.Name] = content
			gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name])
			return
		}
	}
	if v.Union && len(v.MemberTypes) > 0 {
		if _, ok := gen.StructAST[v.Name]; !ok {
			var content string
			for _, member := range toSortedPairs(v.MemberTypes) {
				memberName := member.key
				memberType := member.value

				if memberType == "" { // fix order issue
					memberType = getBasefromSimpleType(memberName, gen.ProtoTree)
				}
				content += genRustFieldCode(v.Name, memberType, false, false, &v.Restriction)
			}
			gen.StructAST[v.Name] = content
			gen.Field += genRustStructCode(genRustStructName(v.Name, true), "", gen.StructAST[v.Name])
		}
		return
	}
	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)
		content := genRustFieldCode(v.Name, fieldType, false, false, &v.Restriction)
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name])
	}
}

// RustComplexType generates code for complex type XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustComplexType(v *ComplexType) {
	var content string
	for _, attrGroup := range v.AttributeGroup {
		fieldType := getBasefromSimpleType(trimNSPrefix(attrGroup.Ref), gen.ProtoTree)
		content += genRustFieldCode(attrGroup.Name, fieldType, false, false, nil)
	}
	for _, attribute := range v.Attributes {
		fieldType := getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)
		content += genRustFieldCode(attribute.Name, fieldType, attribute.Plural, attribute.Optional, nil)
	}
	for _, group := range v.Groups {
		fieldType := getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)
		content += genRustFieldCode(group.Name, fieldType, group.Plural, false, nil)
	}
	for _, element := range v.Elements {
		fieldType := getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)
		content += genRustFieldCode(element.Name, fieldType, element.Plural, element.Optional, nil)
	}
	if len(v.Base) > 0 {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)
		if isRustBuiltInType(v.Base) {
			content += genRustFieldCode("value", fieldType, false, false, nil)
		} else {
			fieldName := genRustFieldName(fieldType)
			// If the type is not a built-in one, add the base type as a nested field tagged with flatten
			content += fmt.Sprintf("\t#[serde(flatten)]\n\tpub %s: %s,\n", fieldName, fieldType)
		}
	}

	if _, ok := gen.StructAST[v.Name]; !ok {
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name])
	} else {
		fmt.Printf("%s\n", content)
	}
}

func isRustBuiltInType(typeName string) bool {
	_, builtIn := rustBuildinType[typeName]
	return builtIn
}

// RustGroup generates code for group XML schema in Rust language syntax.
func (gen *CodeGenerator) RustGroup(v *Group) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		var content string
		for _, element := range v.Elements {
			fieldType := getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)
			content += genRustFieldCode(element.Name, fieldType, element.Plural, element.Optional, &element.Restriction)
		}
		for _, group := range v.Groups {
			fieldType := getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)
			content += genRustFieldCode(group.Name, fieldType, group.Plural, false, nil)
		}
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name])
	}
}

// RustAttributeGroup generates code for attribute group XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustAttributeGroup(v *AttributeGroup) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		var content string
		for _, attribute := range v.Attributes {
			fieldType := getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)
			content += genRustFieldCode(attribute.Name, fieldType, attribute.Plural, attribute.Optional, &attribute.Restriction)
		}
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name])
	}
}

// RustElement generates code for element XML schema in Rust language syntax.
func (gen *CodeGenerator) RustElement(v *Element) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree)
		gen.StructAST[v.Name] = genRustFieldCode(v.Name, fieldType, v.Plural, v.Optional, &v.Restriction)
		gen.Field += genRustStructCode(genRustFieldName(v.Name), v.Doc, gen.StructAST[v.Name])
	}
}

// RustAttribute generates code for attribute XML schema in Rust language syntax.
func (gen *CodeGenerator) RustAttribute(v *Attribute) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree)
		gen.StructAST[v.Name] = genRustFieldCode(v.Name, fieldType, v.Plural, v.Optional, &v.Restriction)
		gen.Field += genRustStructCode(genRustFieldName(v.Name), v.Doc, gen.StructAST[v.Name])
	}
}

// genRustStructName generate struct name for Rust code.
func genRustFieldRename(name string) string {
	if strings.Count(name, ":") > 0 {
		return strings.Split(name, ":")[1]
	} else {
		if name == "value" {
			name = "$" + name
		}
		return name
	}
}

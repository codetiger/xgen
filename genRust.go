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
	commonDerives = "#[cfg_attr(feature = \"derive_default\", derive(Default))]\n#[cfg_attr(feature = \"derive_serde\", derive(Serialize, Deserialize))]\n"
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
	var imports = `
use regex::Regex;
use crate::common::*;
#[cfg(feature = "derive_serde")]
use serde::{Deserialize, Serialize};`
	var extras = `#[cfg_attr(feature = "derive_debug", derive(Debug))]
#[cfg_attr(feature = "derive_clone", derive(Clone))]
#[cfg_attr(feature = "derive_partial_eq", derive(PartialEq))]
`
	source := []byte(fmt.Sprintf("%s\n\n%spub mod %s {%s\n}", copyright, extras, gen.Package, strings.ReplaceAll(imports+gen.Field, "\n", "\n\t")))
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
func genRustFieldCode(name string, ftype string, plural bool, optional bool, restriction *Restriction, untagged bool) (string, string) {
	fieldName := genRustFieldName(name)
	fieldType := genRustFieldType(ftype)
	validations := ""

	if isRustBuiltInType(fieldType) {
		// Only add validation attributes if there are restrictions
		if restriction != nil {
			// Handle minLength
			if restriction.hasMinLength {
				validations += fmt.Sprintf("\nif self.%s.chars().count() < %d {\n", fieldName, restriction.MinLength)
				validations += fmt.Sprintf("return Err(ValidationError::new(1001, \"%s is shorter than the minimum length of %d\".to_string()));\n", fieldName, restriction.MinLength)
				validations += "}"
			}
			// Handle maxLength
			if restriction.hasMaxLength {
				validations += fmt.Sprintf("\nif self.%s.chars().count() > %d {\n", fieldName, restriction.MaxLength)
				validations += fmt.Sprintf("\treturn Err(ValidationError::new(1002, \"%s exceeds the maximum length of %d\".to_string()));\n", fieldName, restriction.MaxLength)
				validations += "}"
			}

			// Handle minInclusive and maxInclusive for numeric types
			if restriction.hasMin {
				validations += fmt.Sprintf("\nif self.%s < %f {\n", fieldName, restriction.Min)
				validations += fmt.Sprintf("\treturn Err(ValidationError::new(1003, \"%s is less than the minimum value of %f\".to_string()));\n", fieldName, restriction.Min)
				validations += "}"
			}
			if restriction.hasMax {
				validations += fmt.Sprintf("\nif self.%s > %f {\n", fieldName, restriction.Max)
				validations += fmt.Sprintf("\treturn Err(ValidationError::new(1004, \"%s exceeds the maximum value of %f\".to_string()));\n", fieldName, restriction.Max)
				validations += "}"
			}

			// Handle pattern constraints for string types
			if restriction.Pattern != nil && fieldType == "String" {
				patternStr := escapeRustString(restriction.Pattern.String())
				validations += fmt.Sprintf("\nlet pattern = Regex::new(\"%s\").unwrap();\n", patternStr)
				validations += fmt.Sprintf("if !pattern.is_match(&self.%s) {\n", fieldName)
				validations += fmt.Sprintf("\treturn Err(ValidationError::new(1005, \"%s does not match the required pattern\".to_string()));\n", fieldName)
				validations += "}"
			}
		}
	} else {
		if optional {
			if plural {
				validations += fmt.Sprintf("\nif let Some(ref %[1]s_vec) = self.%[1]s { for item in %[1]s_vec { if let Err(e) = item.validate() { return Err(e); } } }", fieldName)
			} else {
				validations += fmt.Sprintf("\nif let Some(ref %[1]s_value) = self.%[1]s { if let Err(e) = %[1]s_value.validate() { return Err(e); } }", fieldName)
			}
		} else if plural {
			validations += fmt.Sprintf("\nfor item in &self.%[1]s { if let Err(e) = item.validate() { return Err(e); } }", fieldName)
		} else {
			validations += fmt.Sprintf("\nif let Err(e) = self.%s.validate() { return Err(e); }", fieldName)
		}
	}

	if plural {
		fieldType = "Vec<" + fieldType + ">"
	}

	rename := genRustFieldRename(name)
	if untagged {
		rename = "$value"
	}

	content := ""
	if optional {
		fieldType = "Option<" + fieldType + ">"
		content += fmt.Sprintf("\n#[cfg_attr( feature = \"derive_serde\", serde(rename = \"%s\", skip_serializing_if = \"Option::is_none\") )]\npub %s: %s,", rename, genRustFieldName(name), fieldType)
	} else {
		content += fmt.Sprintf("\n#[cfg_attr( feature = \"derive_serde\", serde(rename = \"%s\") )]\npub %s: %s,", rename, fieldName, fieldType)
	}

	return content, validations
}

func genRustStructCode(name string, doc string, fieldContent string, validations string, untagged bool) string {
	extraTags := ""
	if untagged {
		extraTags += "#[cfg_attr( feature = \"derive_serde\", serde(transparent) )]\n"
	}

	content := fmt.Sprintf("\n%s%s%spub struct %s {%s\n}\n", genFieldComment(name, doc, "//"), commonDerives, extraTags, name, strings.ReplaceAll(fieldContent, "\n", "\n\t"))
	content += fmt.Sprintf("\nimpl %s {\n\tpub fn validate(&self) -> Result<(), ValidationError> {%s\n\t\tOk(())\n\t}\n}\n", name, strings.ReplaceAll(validations, "\n", "\n\t\t"))
	return content
}

func genRustEnumCode(name string, doc string, fieldContent string) string {
	content := fmt.Sprintf("\n%s%spub enum %s {\n\t#[cfg_attr(feature = \"derive_default\", default)]\n", doc, commonDerives, name)
	content += fieldContent
	content += "}\n"
	content += fmt.Sprintf("\nimpl %s {\n\tpub fn validate(&self) -> Result<(), ValidationError> {\n\t\tOk(())\n\t}\n}\n", name)
	return content
}

// RustSimpleType generates code for simple type XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustSimpleType(v *SimpleType) {
	if v.List {
		if _, ok := gen.StructAST[v.Name]; !ok {
			fieldType := getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)
			content, validation := genRustFieldCode(v.Name, fieldType, true, false, &v.Restriction, false)
			gen.StructAST[v.Name] = content
			gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], validation, false)
			return
		}
	}
	if v.Union && len(v.MemberTypes) > 0 {
		if _, ok := gen.StructAST[v.Name]; !ok {
			var content, validation string
			for _, member := range toSortedPairs(v.MemberTypes) {
				memberName := member.key
				memberType := member.value

				if memberType == "" { // fix order issue
					memberType = getBasefromSimpleType(memberName, gen.ProtoTree)
				}
				conts, valids := genRustFieldCode(v.Name, memberType, false, false, &v.Restriction, false)
				content += conts
				validation += valids
			}
			gen.StructAST[v.Name] = content
			gen.Field += genRustStructCode(genRustStructName(v.Name, true), "", gen.StructAST[v.Name], validation, false)
		}
		return
	}
	if len(v.Restriction.Enum) > 0 && v.Base == "String" {
		fieldContent := ""
		for _, enumValue := range v.Restriction.Enum {
			fieldContent += fmt.Sprintf("\t#[cfg_attr( feature = \"derive_serde\", serde(rename = \"%s\") )]\n\tCode%s,\n", enumValue, strings.ToUpper(enumValue))
		}
		gen.StructAST[v.Name] = fieldContent
		enumName := genRustStructName(v.Name, true)
		gen.Field += genRustEnumCode(enumName, genFieldComment(v.Name, v.Doc, "//"), fieldContent)
		return
	}

	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)
		content, validation := genRustFieldCode(v.Name, fieldType, false, false, &v.Restriction, true)
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], validation, true)
	}
}

// RustComplexType generates code for complex type XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustComplexType(v *ComplexType) {
	var content, validation string
	for _, attrGroup := range v.AttributeGroup {
		fieldType := getBasefromSimpleType(trimNSPrefix(attrGroup.Ref), gen.ProtoTree)
		conts, valids := genRustFieldCode(attrGroup.Name, fieldType, false, false, nil, false)
		content += conts
		validation += valids
	}
	for _, attribute := range v.Attributes {
		// fieldType := getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)
		fieldType := "String"
		conts, valids := genRustFieldCode(attribute.Name, fieldType, attribute.Plural, attribute.Optional, nil, false)
		content += conts
		validation += valids
	}
	for _, group := range v.Groups {
		fieldType := getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)
		conts, valids := genRustFieldCode(group.Name, fieldType, group.Plural, false, nil, false)
		content += conts
		validation += valids
	}
	for _, element := range v.Elements {
		fieldType := getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)
		conts, valids := genRustFieldCode(element.Name, fieldType, element.Plural, element.Optional, nil, false)
		content += conts
		validation += valids
	}
	if len(v.Base) > 0 {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Base), gen.ProtoTree)
		if isRustBuiltInType(v.Base) {
			conts, valids := genRustFieldCode("value", fieldType, false, false, nil, false)
			content += conts
			validation += valids
		} else {
			fmt.Printf("\n\n%s\n", fieldType)
			fieldName := genRustFieldName(fieldType)
			// If the type is not a built-in one, add the base type as a nested field tagged with flatten
			content += fmt.Sprintf("\t#[cfg_attr( feature = \"derive_serde\", serde(flatten) )]\n\tpub %s: %s,\n", fieldName, fieldType)
		}
	}

	if _, ok := gen.StructAST[v.Name]; !ok {
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], validation, false)
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
		var content, validation string
		for _, element := range v.Elements {
			fieldType := getBasefromSimpleType(trimNSPrefix(element.Type), gen.ProtoTree)
			conts, valids := genRustFieldCode(element.Name, fieldType, element.Plural, element.Optional, &element.Restriction, false)
			content += conts
			validation += valids
		}
		for _, group := range v.Groups {
			fieldType := getBasefromSimpleType(trimNSPrefix(group.Ref), gen.ProtoTree)
			conts, valids := genRustFieldCode(group.Name, fieldType, group.Plural, false, nil, false)
			content += conts
			validation += valids
		}
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], validation, false)
	}
}

// RustAttributeGroup generates code for attribute group XML schema in Rust language
// syntax.
func (gen *CodeGenerator) RustAttributeGroup(v *AttributeGroup) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		var content, validation string
		for _, attribute := range v.Attributes {
			fieldType := getBasefromSimpleType(trimNSPrefix(attribute.Type), gen.ProtoTree)
			conts, valids := genRustFieldCode(attribute.Name, fieldType, attribute.Plural, attribute.Optional, &attribute.Restriction, false)
			content += conts
			validation += valids
		}
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustStructName(v.Name, true), v.Doc, gen.StructAST[v.Name], validation, false)
	}
}

// RustElement generates code for element XML schema in Rust language syntax.
func (gen *CodeGenerator) RustElement(v *Element) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree)
		content, validation := genRustFieldCode(v.Name, fieldType, v.Plural, v.Optional, &v.Restriction, false)
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustFieldName(v.Name), v.Doc, gen.StructAST[v.Name], validation, false)
	}
}

// RustAttribute generates code for attribute XML schema in Rust language syntax.
func (gen *CodeGenerator) RustAttribute(v *Attribute) {
	if _, ok := gen.StructAST[v.Name]; !ok {
		fieldType := getBasefromSimpleType(trimNSPrefix(v.Type), gen.ProtoTree)
		content, validation := genRustFieldCode(v.Name, fieldType, v.Plural, v.Optional, &v.Restriction, false)
		gen.StructAST[v.Name] = content
		gen.Field += genRustStructCode(genRustFieldName(v.Name), v.Doc, gen.StructAST[v.Name], validation, false)
	}
}

// genRustStructName generate struct name for Rust code.
func genRustFieldRename(name string) string {
	if strings.Count(name, ":") > 0 {
		return strings.Split(name, ":")[1]
	} else {
		if name == "value" {
			name = "$value"
		}
		return name
	}
}

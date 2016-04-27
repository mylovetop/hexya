// Copyright 2016 NDP Systèmes. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"fmt"
	"github.com/npiganeau/yep/yep/orm"
	"github.com/npiganeau/yep/yep/tools"
	"reflect"
	"strings"
)

const (
	defaultStructTagName   = "yep"
	defaultStructTagDelim  = ";"
	defaultDependsTagDelim = ","
)

var (
	supportedTag = map[string]int{
		"store":   1,
		"html":    1,
		"string":  2,
		"help":    2,
		"compute": 2,
		"depends": 2,
	}
)

/*
parseStructTag parses the given structField tag string and fills:
- attrs if the individual tag is boolean
- tags if the individual tag has a string value
*/
func parseStructTag(data string, attrs *map[string]bool, tags *map[string]string) {
	attr := make(map[string]bool)
	tag := make(map[string]string)
	for _, v := range strings.Split(data, defaultStructTagDelim) {
		v = strings.TrimSpace(v)
		if supportedTag[v] == 1 {
			attr[v] = true
		} else if i := strings.Index(v, "("); i > 0 && strings.Index(v, ")") == len(v)-1 {
			name := v[:i]
			if supportedTag[name] == 2 {
				v = v[i+1 : len(v)-1]
				tag[name] = v
			}
		}
	}
	*attrs = attr
	*tags = tag
}

/*
checkStructPtr checks that the given data is a struct ptr valid for receiving data from
the database through the RecordSet API.
It returns an error if it not the case.
*/
func checkStructPtr(data interface{}, allowSlice ...bool) error {
	val := reflect.ValueOf(data)
	ind := reflect.Indirect(val)
	indType := ind.Type()
	sliceAllowed := false
	if len(allowSlice) > 0 {
		sliceAllowed = allowSlice[0]
	}
	if val.Kind() != reflect.Ptr || ind.Kind() != reflect.Struct {
		if ind.Kind() != reflect.Slice || !sliceAllowed {
			return fmt.Errorf("Argument must be a struct pointer")
		} else {
			indType = ind.Type().Elem()
			if indType.Kind() == reflect.Ptr {
				indType = indType.Elem()
			}
			if indType.Kind() != reflect.Struct {
				return fmt.Errorf("Argument must be a slice of structs or slice of struct pointers")
			}
		}
	}

	if _, ok := indType.FieldByName("ID"); !ok {
		return fmt.Errorf("Struct must have an ID field")
	}
	if f, _ := indType.FieldByName("ID"); f.Type != reflect.TypeOf(int64(1.0)) {
		return fmt.Errorf("Struct ID field must be of type `int64`")
	}
	return nil
}

/*
getModelName returns the model name corresponding to the given container
*/
func getModelName(container interface{}) string {
	val := reflect.ValueOf(container)
	ind := reflect.Indirect(val)
	indType := ind.Type()
	if indType.Kind() == reflect.String {
		return container.(string)
	}
	if indType.Kind() == reflect.Slice {
		indType = indType.Elem()
	}
	if indType.Kind() == reflect.Ptr {
		indType = indType.Elem()
	}
	name := strings.SplitN(indType.Name(), "_", 2)[0]
	return name
}

/*
structToMap returns the fields and values of the given struct pointer in a map.
*/
func structToMap(structPtr interface{}) map[string]interface{} {
	val := reflect.ValueOf(structPtr)
	ind := reflect.Indirect(val)
	if val.Kind() != reflect.Ptr || ind.Kind() != reflect.Struct {
		panic(fmt.Errorf("structPtr must be a pointer to a struct"))
	}
	res := make(map[string]interface{})
	for i := 0; i < ind.NumField(); i++ {
		fieldName := ind.Type().Field(i).Name
		fieldValue := ind.Field(i)
		var resVal interface{}
		if fieldValue.Kind() == reflect.Ptr {
			relInd := reflect.Indirect(fieldValue)
			if relInd.Kind() != reflect.Struct {
				continue
			}
			if !relInd.FieldByName("ID").IsValid() {
				continue
			}
			resVal = relInd.FieldByName("ID").Interface()
		} else {
			resVal = ind.Field(i).Interface()
		}
		res[fieldName] = resVal
	}
	return res
}

/*
getFieldType returns the FieldType of the given ref field
*/
func getFieldType(ref fieldRef) tools.FieldType {
	fi := orm.FieldGet(ref.modelName, ref.name)
	fc, _ := fieldsCache.get(ref)
	switch fi.FieldType {
	case orm.TypeBooleanField:
		return tools.BOOLEAN
	case orm.TypeCharField:
		return tools.CHAR
	case orm.TypeTextField:
		if fc.html {
			return tools.HTML
		} else {
			return tools.TEXT
		}
	case orm.TypeDateField:
		return tools.DATE
	case orm.TypeDateTimeField:
		return tools.DATETIME
	case orm.TypeFloatField:
		return tools.FLOAT
	case orm.TypeDecimalField:
		return tools.FLOAT
	case orm.RelForeignKey:
		return tools.MANY2ONE
	case orm.RelOneToOne:
		return tools.ONE2ONE
	case orm.RelReverseOne:
		return tools.ONE2MANY
	case orm.RelManyToMany:
		return tools.MANY2MANY
	case orm.RelReverseMany:
		return tools.NO_TYPE
	}
	switch {
	case fi.FieldType&orm.IsIntegerField > 0:
		return tools.INTEGER
	}
	return tools.NO_TYPE
}

// GetFieldColumn returns the column name of the given field of the
// given model.
// If name is already a column name return it anyway.
func GetFieldColumn(model, name string) string {
	fi := orm.FieldGet(model, name)
	return fi.Column
}

// ConvertFieldName returns the field name of the given model with
// the given column. If column is already a field name, returns it
// anyway.
func GetFieldName(model, column string) string {
	ref := fieldRef{modelName: model, name: column}
	ref.ConvertToName()
	return ref.name
}
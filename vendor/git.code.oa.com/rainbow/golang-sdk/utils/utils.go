package utils

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"git.code.oa.com/rainbow/golang-sdk/keep"
	"gopkg.in/yaml.v2"
)

func parseDatetime(bts []byte, v interface{}) (err error) {
	*(v.(*time.Time)), err = time.Parse("2006-01-02", string(bts))
	if err != nil {
		return
	}
	return
}

var parserMap = map[string]func([]byte, interface{}) error{
	"xml":      xml.Unmarshal,
	"json":     json.Unmarshal,
	"yaml":     yaml.Unmarshal,
	"datetime": parseDatetime,
}

func ConvertGroupToStruct(group keep.Group, rconfig interface{}) (err error) {
	var (
		fieldVal reflect.Value
		tmp      interface{}
	)
	fVal := reflect.ValueOf(rconfig).Elem()
	fTyp := fVal.Type()
	if fTyp.Kind() != reflect.Struct {
		return fmt.Errorf("typ: %s must be struct", fTyp)
	}
	numFields := fTyp.NumField()
	for i := 0; i < numFields; i++ {
		fieldVal = fVal.Field(i)
		rainbowKey, ok := fTyp.Field(i).Tag.Lookup("json")
		if !ok || rainbowKey == "" {
			continue
		}
		typ := fTyp.Field(i).Tag.Get("type")
		knd := fTyp.Field(i).Type.Kind()
		if typ == "" && knd == reflect.Ptr {
			err = ConvertGroupToStruct(group, fieldVal.Interface())
			if err != nil {
				return
			}
			continue
		}
		if typ == "" && knd == reflect.Struct {
			err = ConvertGroupToStruct(group, fieldVal.Addr().Interface())
			if err != nil {
				return
			}
			continue
		}
		tmpStr, ok := group[rainbowKey].(string)
		if !ok {
			continue
		}
		if fieldVal.Type().Kind() == reflect.Ptr {
			fieldVal = fieldVal.Elem()
		}
		switch fTyp.Field(i).Type.Kind() {
		case reflect.Int, reflect.Int16, reflect.Int32, reflect.Int64:
			tmp, err = strconv.ParseInt(tmpStr, 10, 64)
			if err != nil {
				return
			}
			fVal.Field(i).SetInt(tmp.(int64))
		case reflect.Uint, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			tmp, err = strconv.ParseUint(tmpStr, 10, 64)
			if err != nil {
				return
			}
			fVal.Field(i).SetUint(tmp.(uint64))
		case reflect.Float32, reflect.Float64:
			tmp, err = strconv.ParseFloat(tmpStr, 64)
			if err != nil {
				return
			}
			fVal.Field(i).SetFloat(tmp.(float64))
		case reflect.Bool:
			tmp, err = strconv.ParseBool(tmpStr)
			if err != nil {
				return
			}
			fVal.Field(i).SetBool(tmp.(bool))
		case reflect.String:
			fVal.Field(i).SetString(tmpStr)
		case reflect.Slice, reflect.Struct, reflect.Map, reflect.Ptr:
			if typ == "" {
				typ = "json" // default json type
			}
			tmpVal := reflect.New(fieldVal.Type())
			if err = parserMap[typ]([]byte(tmpStr), tmpVal.Interface()); err != nil {
				return
			}
			fieldVal.Set(tmpVal.Elem())
		default:
			err = fmt.Errorf("name: %s, unknown typ: %v",
				fTyp.Field(i).Name, fTyp.Field(i).Type.Kind())
			return
		}
	}
	return
}

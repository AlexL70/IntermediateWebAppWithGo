package models

import (
	"fmt"
	"reflect"
)

func modelCopy(dest, src any) error {
	dType := reflect.TypeOf(dest)
	idType := reflect.Indirect(reflect.ValueOf(dest)).Type()
	sType := reflect.TypeOf(src)

	if dType != sType && idType != sType {
		return fmt.Errorf("cannot copy %v to %v; incompatible types", src, dest)
	}

	switch dType.Kind() {
	case reflect.Array:
	case reflect.Slice:
		_ = reflect.Copy(reflect.ValueOf(dest), reflect.ValueOf(src))
		return nil
	case reflect.Pointer:
	case reflect.Struct:
		break
	default:
		return fmt.Errorf("cannot copy to non-assignable destination")
	}

	dElem := reflect.ValueOf(dest).Elem()
	if dElem.Kind() == reflect.Struct {
		sElem := reflect.ValueOf(src)
		if sElem.Kind() == reflect.Pointer {
			sElem = sElem.Elem()
		}
		for i := 0; i < dElem.NumField(); i++ {
			tag := idType.Field(i).Tag
			if tag.Get("model-copy") == "ignore" {
				continue
			}
			f := idType.Field(i)
			k := f.Type.Kind()
			switch k {
			case reflect.Struct:
				modelCopy(dElem.Field(i).Addr().Interface(), sElem.Field(i).Interface())
			case reflect.Bool:
				dElem.Field(i).SetBool(sElem.Field(i).Bool())
			case reflect.String:
				dElem.Field(i).SetString(sElem.Field(i).String())
			case reflect.Int:
			case reflect.Int8:
			case reflect.Int16:
			case reflect.Int32:
			case reflect.Int64:
				dElem.Field(i).SetInt(sElem.Field(i).Int())
			case reflect.Float32:
			case reflect.Float64:
				dElem.Field(i).SetFloat(sElem.Field(i).Float())
			}
		}
	} else {
		return fmt.Errorf("copying of %s not implemented yet", dElem.Type())
	}

	return nil
}

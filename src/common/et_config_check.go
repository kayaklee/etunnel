package common

import (
	"fmt"
	"reflect"
)

var configCheckStrategy = map[string](func(string, interface{}) error){}

func init() {
	configCheckStrategy = map[string](func(string, interface{}) error){
		"NOP":                 checkNone,
		"IntNotZero":          checkIntNotZero,
		"IntGTZero":           checkIntGTZero,
		"IntSliceGTZero":      checkIntSliceGTZero,
		"StringNotEmpty":      checkStringNotEmpty,
		"StringSliceNotEmpty": checkStringSliceNotEmpty,
		"StructSliceNotEmpty": checkStructSliceNotEmpty,
		"Struct":              configCheckStruct,
	}
}

func checkNone(name string, d interface{}) (err error) {
	err = nil
	return err
}

func checkIntNotZero(name string, d interface{}) (err error) {
	v := reflect.ValueOf(d)
	if v.Int() == 0 {
		err = fmt.Errorf("Invalid %s=%d", name, v.Int())
	}
	return err
}

func checkIntGTZero(name string, d interface{}) (err error) {
	v := reflect.ValueOf(d)
	if v.Int() <= 0 {
		err = fmt.Errorf("Invalid %s=%d", name, v.Int())
	}
	return err
}

func checkIntSliceGTZero(name string, d interface{}) (err error) {
	v := reflect.ValueOf(d)
	if v.Len() <= 0 {
		err = fmt.Errorf("Invalid %s=[%d]", name, v.Interface().([]int64))
	} else {
		ss := v.Interface().([]int64)
		for i, s := range ss {
			if s <= 0 {
				err = fmt.Errorf("Invalid %s[%d]=[%d]", name, i, s)
				break
			}
		}
	}
	return err
}

func checkStringNotEmpty(name string, d interface{}) (err error) {
	v := reflect.ValueOf(d)
	if v.String() == "" {
		err = fmt.Errorf("Invalid %s=[%s]", name, v.String())
	}
	return err
}

func checkStringSliceNotEmpty(name string, d interface{}) (err error) {
	v := reflect.ValueOf(d)
	if v.Len() <= 0 {
		err = fmt.Errorf("Invalid %s=[%s]", name, v.Interface().([]string))
	} else {
		ss := v.Interface().([]string)
		for i, s := range ss {
			if s == "" {
				err = fmt.Errorf("Invalid %s[%d]=[%s]", name, i, s)
				break
			}
		}
	}
	return err
}

func checkStructSliceNotEmpty(name string, d interface{}) (err error) {
	v := reflect.ValueOf(d)
	if v.Len() <= 0 {
		err = fmt.Errorf("Invalid %s=[%v]", name, v.Interface())
	} else {
		for i := 0; i < v.Len() && err == nil; i++ {
			err = configCheckStruct(fmt.Sprintf("%s[%d]", name, i), v.Index(i).Interface())
		}
	}
	return err
}

func configCheckStruct(host string, d interface{}) (err error) {
	t := reflect.TypeOf(d)
	v := reflect.ValueOf(d)
	for i := 0; i < v.NumField() && err == nil; i++ {
		tfield := t.Field(i)
		vfield := v.Field(i)
		check_strategy := tfield.Tag.Get("check")
		check_func, exist := configCheckStrategy[check_strategy]
		if exist {
			err = check_func(host+"."+tfield.Name, vfield.Interface())
		}
	}
	return err
}

func configStringStruct(host string, d interface{}) (ret string) {
	t := reflect.TypeOf(d)
	v := reflect.ValueOf(d)
	for i := 0; i < v.NumField(); i++ {
		tfield := t.Field(i)
		vfield := v.Field(i)
		if !vfield.CanInterface() {
			continue
		}
		if reflect.Struct == vfield.Kind() {
			ret += configStringStruct(host+"."+tfield.Name, vfield.Interface())
		} else {
			ret += fmt.Sprintf("\n\t%s=%v", host+"."+tfield.Name, vfield.Interface())
		}
	}
	return ret
}

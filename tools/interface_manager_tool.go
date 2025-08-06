package tools

import "reflect"

func CopyStructFields(src interface{}, dst interface{}) {
	srcVal := reflect.ValueOf(src).Elem()
	dstVal := reflect.ValueOf(dst).Elem()

	for i := 0; i < srcVal.NumField(); i++ {
		field := srcVal.Type().Field(i)
		srcFieldVal := srcVal.FieldByName(field.Name)
		dstFieldVal := dstVal.FieldByName(field.Name)

		if dstFieldVal.IsValid() && dstFieldVal.CanSet() && srcFieldVal.Type() == dstFieldVal.Type() {
			dstFieldVal.Set(srcFieldVal)
		}
	}
}

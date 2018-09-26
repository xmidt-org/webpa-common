package xerrors

import (
	"errors"
	"fmt"
	"reflect"
)

func getError(unknown interface{}) error {
	return findErrorInValue(reflect.ValueOf(unknown))
}

// findErrorInValue will try and genereate and error object either from the object itself if it can be created or its
// fields.
func findErrorInValue(v reflect.Value) error {
	if v.Kind() == reflect.Invalid {
		return nil
	}

	if v.CanInterface() {
		// the object can be created, hopefully we can cast it to an error.
		if _, ok := v.Interface().(*error); ok {
			return findErrorInValue(v.Elem())
		} else if err, ok := v.Interface().(error); ok {
			if v.Type().String() == "*errors.errorString" {
				return err
			}
			if v.Kind() == reflect.Interface {
				return err
			} else if v.Kind() == reflect.Ptr {
				return findErrorInValue(v.Elem())
			}
			return err
		} else if v.Kind() == reflect.Ptr { // We can't cast it to an error, check the pointer.
			return findErrorInValue(v.Elem())
		} else if v.Kind() == reflect.Struct {// We can't cast it to an error, check the fields.
			return iterateFields(v)
		}
	} else if v.Kind() == reflect.Ptr && v.Type().String() == "*errors.errorString" { // the base case. AKA errors.New("error")
		return errors.New(fmt.Sprintf("%s", v.Elem().Field(0)))
	} else if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface { // try what the object contains.
		return findErrorInValue(v.Elem())
	} else if v.Kind() == reflect.Struct { // try the fields
		return iterateFields(v)
	}

	// an error could not be created
	return nil
}

// FirstCause will find the first cause of the error. This is making the assumption that the errors are cascaded.
// aka an Error struct will contain an error and so on.
func FirstCause(err error) error {
	return findRootError(err)
}

// findRootError will iterate over all the fields and the fields of those fields to generate an error. If no error is
// found <nil> will be returned
func findRootError(unknown interface{}) error {
	if unknown == nil {
		return nil
	}

	v := reflect.ValueOf(unknown)
	currentLevel := findErrorInValue(v)
	var tempErr error

	if v.Kind() == reflect.Ptr { // check if itself and then the fields
		e := findErrorInValue(v.Elem())
		if e != nil {
			e = findRootError(e)
			if e != nil {
				tempErr = e
			}
		}
	} else if v.Kind() == reflect.Struct { // check the fields
		e := iterateFields(v)
		if e != nil {
			e = findRootError(e)
			if e != nil {
				tempErr = e
			}
		}
	}

	if tempErr != nil {
		return tempErr
	}

	return currentLevel
}

// iterateFields iterates over the variables of the struct and tries to create an error object from the field.
// :warning: only can be called where v.Kind() == reflect.Struct
func iterateFields(v reflect.Value) error {
	var tempErr error

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		e := findErrorInValue(f)
		if e != nil {
			tempErr = e
		}
		if f.CanInterface() { // try and create an error if we can turn it into an interface{}
			e = findRootError(f.Interface())
			if e != nil {
				tempErr = e
			}
		} else if f.Kind() == reflect.Interface { // try the object the field contains
			f = f.Elem()
			e = findErrorInValue(f)
			if e != nil {
				tempErr = e
			}
		}
	}

	return tempErr
}

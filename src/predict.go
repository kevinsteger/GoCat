package main

/*
#cgo linux LDFLAGS: -L${SRCDIR}/lib -Wl,-rpath,$ORIGIN/lib -lcatboostmodel
#include <stdlib.h>
#include <stdbool.h>
#include "lib/c_api.h"

static char** makeCharArray(int size) {
        return calloc(sizeof(char*), size);
}

static void setArrayString(char **a, char *s, int n) {
        a[n] = s;
}

static void freeCharArray(char **a, int size) {
        int i;
        for (i = 0; i < size; i++)
                free(a[i]);
        free(a);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type Model struct {
	Handler unsafe.Pointer
	UUID    string
}

func LoadModel(filename string) (*Model, error) {
	model := &Model{}
	model.Handler = C.ModelCalcerCreate()
	if !C.LoadFullModelFromFile(model.Handler, C.CString(filename)) {
		return nil, fmt.Errorf(C.GoString(C.GetErrorString()))
	}
	return model, nil
}

func (model *Model) GetPrediction(inputArray []interface{}) (float64, error) {
	cats := []string{}
	floats := []float32{}

	//since the input array can contain both strings and floats, separate the two
	//and append to their corresponding slices
	for _, element := range inputArray {
		switch v := element.(type) {
		case string:
			cats = append(cats, element.(string))
		case float32:
			floats = append(floats, element.(float32))
		case float64:
			floats = append(floats, float32(element.(float64)))
		case int:
			floats = append(floats, float32(element.(int)))
		default:
			fmt.Printf("error: unexpected type %T", v)
		}
	}

	var result float64
	floatLength := len(floats)
	catLength := len(cats)

	//go slice to C array
	pointer := makeCStringArrayPointer(cats)
	defer C.freeCharArray(pointer, C.int(len(cats)))
	catsC := pointer

	//make prediction for single array
	if !C.CalcModelPredictionSingle(
		model.Handler,
		(*C.float)(&floats[0]),
		C.size_t(floatLength),
		(**C.char)(catsC),
		C.size_t(catLength),
		(*C.double)(&result),
		C.size_t(1.0),
	) {
		return 0.0, getError()
	}

	return result, nil
}

//helper functions to interact w C
func getError() error {
	messageC := C.GetErrorString()
	message := C.GoString(messageC)
	return fmt.Errorf(message)
}

func makeCStringArrayPointer(array []string) **C.char {
	cargs := C.makeCharArray(C.int(len(array)))
	for i, s := range array {
		C.setArrayString(cargs, C.CString(s), C.int(i))
	}
	return cargs
}

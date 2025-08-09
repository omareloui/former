// Package former provides HTTP form data binding to Go structs using struct tags.
//
// Former simplifies the process of populating Go structs from HTTP form data by using
// reflection and struct field tags. It supports both application/x-www-form-urlencoded
// and multipart/form-data content types.
//
// # Basic Usage
//
// Define a struct with formfield tags and use Populate to fill it from an HTTP request:
//
//	type LoginForm struct {
//		Username string `formfield:"username"`
//		Password string `formfield:"password"`
//		Remember bool   `formfield:"remember"`
//	}
//
//	func handler(w http.ResponseWriter, r *http.Request) {
//		var form LoginForm
//		if err := former.Populate(r, &form); err != nil {
//			http.Error(w, err.Error(), http.StatusBadRequest)
//			return
//		}
//		// form is now populated with values from the request
//	}
//
// # Supported Types
//
// Former supports all basic Go types and many complex types:
//
//   - Basic types: string, bool, int*, uint*, float32, float64
//   - Slices: []string, []int, etc. (multiple form values with same name)
//   - Arrays: [N]T (fills up to array capacity)
//   - Maps: map[string]string (expects "key:value" format)
//   - Pointers: *T (automatically initialized if values are present)
//   - Structs: nested structs with their own formfield tags
//
// # Nested Structures
//
// Former handles nested structures in multiple ways:
//
// 1. Embedded structs (no tag) - fields are treated as top-level:
//
//	type Person struct {
//		Name    string `formfield:"name"`
//		Address        // embedded
//	}
//	type Address struct {
//		Street string `formfield:"street"`
//		City   string `formfield:"city"`
//	}
//	// Form data: name=John&street=Main St&city=NYC
//
// 2. Nested structs with tags - can use dot notation:
//
//	type Order struct {
//		Shipping Address `formfield:"shipping"`
//	}
//	// Form data: shipping.street=Main St&shipping.city=NYC
//
// 3. Nested structs as JSON:
//
//	type User struct {
//		Profile Profile `formfield:"profile"`
//	}
//	// Form data: profile={"age":30,"bio":"Developer"}
//
// # Special Features
//
// - Fields with tag `formfield:"-"` are skipped
// - Checkbox values "on", "1", and "true" are treated as true for bool fields
// - File uploads can be retrieved using GetFile function
//
// # Error Handling
//
// Former follows these error handling principles:
// - Type conversion errors are returned immediately
// - Invalid JSON in struct fields returns an error
// - The target must be a pointer to a struct
//
// # Multipart Forms
//
// For file uploads, use multipart/form-data encoding:
//
//	file, header, err := former.GetFile(r, "upload")
//	if err != nil {
//		// handle error
//	}
//	defer file.Close()
package former

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

func Populate(r *http.Request, dest any) error {
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Ptr || rv.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to a struct")
	}

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); // 32MB max memory
		err != nil {
			return fmt.Errorf("failed to parse multipart form: %w", err)
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return fmt.Errorf("failed to parse form: %w", err)
		}
	}

	structValue := rv.Elem()
	structType := structValue.Type()

	return populateStruct(structValue, structType, r, "")
}

func populateStruct(structValue reflect.Value, structType reflect.Type, r *http.Request, prefix string) error {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)

		if !fieldValue.CanSet() {
			continue
		}

		formFieldName := field.Tag.Get("formfield")

		if formFieldName == "" {
			if field.Anonymous && fieldValue.Kind() == reflect.Struct {
				if err := populateStruct(fieldValue, fieldValue.Type(), r, prefix); err != nil {
					return err
				}
			}
			continue
		}

		if formFieldName == "-" {
			continue
		}

		fullFieldName := formFieldName
		if prefix != "" {
			fullFieldName = prefix + "." + formFieldName
		}

		if fieldValue.Kind() == reflect.Struct {
			if values := getFormValues(r, fullFieldName); len(values) > 0 {
				jsonLike := looksLikeJSON(values[0])
				if jsonLike {
					if err := json.Unmarshal([]byte(values[0]), fieldValue.Addr().Interface()); err != nil {
						return fmt.Errorf("failed to parse JSON for field %s: %w", field.Name, err)
					}
					continue
				}
			}

			if err := populateStruct(fieldValue, fieldValue.Type(), r, fullFieldName); err != nil {
				return err
			}
			continue
		}

		if fieldValue.Kind() == reflect.Ptr {
			hasValues := false

			if values := getFormValues(r, fullFieldName); len(values) > 0 {
				hasValues = true
			} else if fieldValue.Type().Elem().Kind() == reflect.Struct {
				elemType := fieldValue.Type().Elem()
				for j := 0; j < elemType.NumField(); j++ {
					nestedField := elemType.Field(j)
					nestedTag := nestedField.Tag.Get("formfield")
					if nestedTag != "" && nestedTag != "-" {
						nestedName := fullFieldName + "." + nestedTag
						if values := getFormValues(r, nestedName); len(values) > 0 {
							hasValues = true
							break
						}
					}
				}
			}

			if hasValues {
				if fieldValue.IsNil() {
					fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
				}

				if fieldValue.Elem().Kind() == reflect.Struct {
					if err := populateStruct(fieldValue.Elem(), fieldValue.Elem().Type(), r, fullFieldName); err != nil {
						return err
					}
				} else {
					if values := getFormValues(r, fullFieldName); len(values) > 0 {
						if err := setFieldValue(fieldValue.Elem(), values); err != nil {
							return fmt.Errorf("failed to set field %s: %w", field.Name, err)
						}
					}
				}
			}
			continue
		}

		values := getFormValues(r, fullFieldName)
		if len(values) == 0 {
			if prefix != "" {
				values = getFormValues(r, formFieldName)
			}
			if len(values) == 0 {
				continue
			}
		}

		if err := setFieldValue(fieldValue, values); err != nil {
			return fmt.Errorf("failed to set field %s: %w", field.Name, err)
		}
	}

	return nil
}

func getFormValues(r *http.Request, fieldName string) []string {
	if values, ok := r.Form[fieldName]; ok {
		return values
	}

	if r.MultipartForm != nil {
		if values, ok := r.MultipartForm.Value[fieldName]; ok {
			return values
		}
	}

	return nil
}

func setFieldValue(fieldValue reflect.Value, values []string) error {
	fieldType := fieldValue.Type()

	switch fieldType.Kind() {
	case reflect.String:
		if len(values) > 0 {
			fieldValue.SetString(values[0])
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if len(values) > 0 {
			intVal, err := strconv.ParseInt(values[0], 10, fieldType.Bits())
			if err != nil {
				return err
			}
			fieldValue.SetInt(intVal)
		}

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if len(values) > 0 {
			uintVal, err := strconv.ParseUint(values[0], 10, fieldType.Bits())
			if err != nil {
				return err
			}
			fieldValue.SetUint(uintVal)
		}

	case reflect.Float32, reflect.Float64:
		if len(values) > 0 {
			floatVal, err := strconv.ParseFloat(values[0], fieldType.Bits())
			if err != nil {
				return err
			}
			fieldValue.SetFloat(floatVal)
		}

	case reflect.Bool:
		if len(values) > 0 {
			boolVal, err := strconv.ParseBool(values[0])
			if err != nil {
				boolVal = values[0] == "on" || values[0] == "1" || values[0] == "true"
			}
			fieldValue.SetBool(boolVal)
		}

	case reflect.Slice:
		return setSliceValue(fieldValue, values)

	case reflect.Array:
		return setArrayValue(fieldValue, values)

	case reflect.Map:
		return setMapValue(fieldValue, values)

	case reflect.Ptr:
		if len(values) > 0 {
			if fieldValue.IsNil() {
				fieldValue.Set(reflect.New(fieldType.Elem()))
			}
			return setFieldValue(fieldValue.Elem(), values)
		}

	case reflect.Struct:
		log.Panic("struct fields should be handled in populateStruct")

	default:
		return fmt.Errorf("unsupported field type: %s", fieldType.Kind())
	}

	return nil
}

func setSliceValue(fieldValue reflect.Value, values []string) error {
	sliceType := fieldValue.Type()

	newSlice := reflect.MakeSlice(sliceType, len(values), len(values))

	for i, value := range values {
		elem := newSlice.Index(i)
		if err := setFieldValue(elem, []string{value}); err != nil {
			return err
		}
	}

	fieldValue.Set(newSlice)
	return nil
}

func setArrayValue(fieldValue reflect.Value, values []string) error {
	arrayLen := fieldValue.Len()

	for i := 0; i < arrayLen && i < len(values); i++ {
		elem := fieldValue.Index(i)
		if err := setFieldValue(elem, []string{values[i]}); err != nil {
			return err
		}
	}

	return nil
}

func setMapValue(fieldValue reflect.Value, values []string) error {
	mapType := fieldValue.Type()
	keyType := mapType.Key()
	valueType := mapType.Elem()

	newMap := reflect.MakeMap(mapType)

	for _, value := range values {
		parts := strings.SplitN(value, ":", 2)
		if len(parts) != 2 {
			continue
		}

		keyVal := reflect.New(keyType).Elem()
		if err := setFieldValue(keyVal, []string{parts[0]}); err != nil {
			return err
		}

		valVal := reflect.New(valueType).Elem()
		if err := setFieldValue(valVal, []string{parts[1]}); err != nil {
			return err
		}

		newMap.SetMapIndex(keyVal, valVal)
	}

	fieldValue.Set(newMap)
	return nil
}

func looksLikeJSON(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}")) ||
		(strings.HasPrefix(s, "[") && strings.HasSuffix(s, "]"))
}

func GetFile(r *http.Request, fieldName string) (multipart.File, *multipart.FileHeader, error) {
	if r.MultipartForm == nil {
		return nil, nil, fmt.Errorf("no multipart form data")
	}

	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return nil, nil, err
	}

	return file, header, nil
}

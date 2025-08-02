package former

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

type BasicTypes struct {
	String  string  `formfield:"string"`
	Int     int     `formfield:"int"`
	Int8    int8    `formfield:"int8"`
	Int16   int16   `formfield:"int16"`
	Int32   int32   `formfield:"int32"`
	Int64   int64   `formfield:"int64"`
	Uint    uint    `formfield:"uint"`
	Uint8   uint8   `formfield:"uint8"`
	Uint16  uint16  `formfield:"uint16"`
	Uint32  uint32  `formfield:"uint32"`
	Uint64  uint64  `formfield:"uint64"`
	Float32 float32 `formfield:"float32"`
	Float64 float64 `formfield:"float64"`
	Bool    bool    `formfield:"bool"`
}

type ComplexTypes struct {
	Slice      []string          `formfield:"slice"`
	IntSlice   []int             `formfield:"intslice"`
	Array      [3]string         `formfield:"array"`
	Map        map[string]string `formfield:"map"`
	MapIntVal  map[string]int    `formfield:"mapint"`
	PointerStr *string           `formfield:"ptrstr"`
	PointerInt *int              `formfield:"ptrint"`
}

type Address struct {
	Street  string `formfield:"street"`
	City    string `formfield:"city"`
	ZipCode string `formfield:"zip"`
}

type Contact struct {
	Phone string `formfield:"phone"`
	Email string `formfield:"email"`
}

type Person struct {
	Name string `formfield:"name"`
	Age  int    `formfield:"age"`
	Address
	Contact Contact `formfield:"contact"`
}

type Profile struct {
	Bio      string   `formfield:"bio"`
	Hobbies  []string `formfield:"hobbies"`
	Settings struct {
		Theme    string `formfield:"theme" json:"theme"`
		Language string `formfield:"lang" json:"lang"`
	} `formfield:"settings"`
}

type StructWithSkippedField struct {
	Public  string `formfield:"public"`
	Skipped string `formfield:"-"`
	NoTag   string
}

func TestPopulate_BasicTypes(t *testing.T) {
	tests := []struct {
		name     string
		formData url.Values
		expected BasicTypes
	}{
		{
			name: "all basic types",
			formData: url.Values{
				"string":  {"hello world"},
				"int":     {"42"},
				"int8":    {"127"},
				"int16":   {"32767"},
				"int32":   {"2147483647"},
				"int64":   {"9223372036854775807"},
				"uint":    {"42"},
				"uint8":   {"255"},
				"uint16":  {"65535"},
				"uint32":  {"4294967295"},
				"uint64":  {"18446744073709551615"},
				"float32": {"3.14"},
				"float64": {"3.141592653589793"},
				"bool":    {"true"},
			},
			expected: BasicTypes{
				String:  "hello world",
				Int:     42,
				Int8:    127,
				Int16:   32767,
				Int32:   2147483647,
				Int64:   9223372036854775807,
				Uint:    42,
				Uint8:   255,
				Uint16:  65535,
				Uint32:  4294967295,
				Uint64:  18446744073709551615,
				Float32: 3.14,
				Float64: 3.141592653589793,
				Bool:    true,
			},
		},
		{
			name: "bool variations",
			formData: url.Values{
				"bool": {"on"},
			},
			expected: BasicTypes{
				Bool: true,
			},
		},
		{
			name: "bool checkbox value",
			formData: url.Values{
				"bool": {"1"},
			},
			expected: BasicTypes{
				Bool: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			var result BasicTypes
			err := Populate(req, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestPopulate_ComplexTypes(t *testing.T) {
	strPtr := "pointer value"
	intPtr := 42

	tests := []struct {
		name     string
		formData url.Values
		expected ComplexTypes
	}{
		{
			name: "slices and arrays",
			formData: url.Values{
				"slice":    {"value1", "value2", "value3"},
				"intslice": {"1", "2", "3"},
				"array":    {"a", "b", "c"},
			},
			expected: ComplexTypes{
				Slice:    []string{"value1", "value2", "value3"},
				IntSlice: []int{1, 2, 3},
				Array:    [3]string{"a", "b", "c"},
			},
		},
		{
			name: "maps",
			formData: url.Values{
				"map":    {"key1:value1", "key2:value2"},
				"mapint": {"one:1", "two:2", "three:3"},
			},
			expected: ComplexTypes{
				Map:       map[string]string{"key1": "value1", "key2": "value2"},
				MapIntVal: map[string]int{"one": 1, "two": 2, "three": 3},
			},
		},
		{
			name: "pointers",
			formData: url.Values{
				"ptrstr": {"pointer value"},
				"ptrint": {"42"},
			},
			expected: ComplexTypes{
				PointerStr: &strPtr,
				PointerInt: &intPtr,
			},
		},
		{
			name: "array overflow",
			formData: url.Values{
				"array": {"a", "b", "c", "d", "e"},
			},
			expected: ComplexTypes{
				Array: [3]string{"a", "b", "c"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			var result ComplexTypes
			err := Populate(req, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result.Slice, tt.expected.Slice) {
				t.Errorf("Slice: got %v, want %v", result.Slice, tt.expected.Slice)
			}
			if !reflect.DeepEqual(result.IntSlice, tt.expected.IntSlice) {
				t.Errorf("IntSlice: got %v, want %v", result.IntSlice, tt.expected.IntSlice)
			}
			if !reflect.DeepEqual(result.Array, tt.expected.Array) {
				t.Errorf("Array: got %v, want %v", result.Array, tt.expected.Array)
			}

			if !reflect.DeepEqual(result.Map, tt.expected.Map) {
				t.Errorf("Map: got %v, want %v", result.Map, tt.expected.Map)
			}
			if !reflect.DeepEqual(result.MapIntVal, tt.expected.MapIntVal) {
				t.Errorf("MapIntVal: got %v, want %v", result.MapIntVal, tt.expected.MapIntVal)
			}

			if tt.expected.PointerStr != nil {
				if result.PointerStr == nil || *result.PointerStr != *tt.expected.PointerStr {
					t.Errorf("PointerStr: got %v, want %v", result.PointerStr, tt.expected.PointerStr)
				}
			}
			if tt.expected.PointerInt != nil {
				if result.PointerInt == nil || *result.PointerInt != *tt.expected.PointerInt {
					t.Errorf("PointerInt: got %v, want %v", result.PointerInt, tt.expected.PointerInt)
				}
			}
		})
	}
}

func TestPopulate_NestedStructs(t *testing.T) {
	tests := []struct {
		name     string
		formData url.Values
		expected Person
	}{
		{
			name: "embedded struct fields",
			formData: url.Values{
				"name":   {"John Doe"},
				"age":    {"30"},
				"street": {"123 Main St"},
				"city":   {"New York"},
				"zip":    {"10001"},
			},
			expected: Person{
				Name: "John Doe",
				Age:  30,
				Address: Address{
					Street:  "123 Main St",
					City:    "New York",
					ZipCode: "10001",
				},
			},
		},
		{
			name: "nested struct as JSON",
			formData: url.Values{
				"name":    {"Jane Doe"},
				"age":     {"25"},
				"contact": {`{"phone":"555-1234","email":"jane@example.com"}`},
			},
			expected: Person{
				Name: "Jane Doe",
				Age:  25,
				Contact: Contact{
					Phone: "555-1234",
					Email: "jane@example.com",
				},
			},
		},
		{
			name: "nested struct with dot notation",
			formData: url.Values{
				"name":          {"Bob Smith"},
				"age":           {"35"},
				"contact.phone": {"555-5678"},
				"contact.email": {"bob@example.com"},
			},
			expected: Person{
				Name: "Bob Smith",
				Age:  35,
				Contact: Contact{
					Phone: "555-5678",
					Email: "bob@example.com",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			var result Person
			err := Populate(req, &result)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("got %+v, want %+v", result, tt.expected)
			}
		})
	}
}

func TestPopulate_ComplexNestedStructs(t *testing.T) {
	formData := url.Values{
		"bio":            {"Software developer"},
		"hobbies":        {"coding", "reading", "gaming"},
		"settings":       {`{"theme":"dark","lang":"en"}`},
		"settings.theme": {"light"},
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var result Profile
	err := Populate(req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := Profile{
		Bio:     "Software developer",
		Hobbies: []string{"coding", "reading", "gaming"},
	}
	expected.Settings.Theme = "dark"
	expected.Settings.Language = "en"

	if result.Bio != expected.Bio {
		t.Errorf("Bio: got %v, want %v", result.Bio, expected.Bio)
	}
	if !reflect.DeepEqual(result.Hobbies, expected.Hobbies) {
		t.Errorf("Hobbies: got %v, want %v", result.Hobbies, expected.Hobbies)
	}
	if result.Settings.Theme != "dark" {
		t.Errorf("Settings.Theme: got %v, want 'dark'", result.Settings.Theme)
	}
}

func TestPopulate_MultipartForm(t *testing.T) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	w.WriteField("string", "multipart value")
	w.WriteField("int", "123")
	w.WriteField("bool", "true")
	w.WriteField("slice", "val1")
	w.WriteField("slice", "val2")

	fw, _ := w.CreateFormFile("file", "test.txt")
	fw.Write([]byte("file content"))

	w.Close()

	req := httptest.NewRequest("POST", "/", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	var result struct {
		String string   `formfield:"string"`
		Int    int      `formfield:"int"`
		Bool   bool     `formfield:"bool"`
		Slice  []string `formfield:"slice"`
	}

	err := Populate(req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := struct {
		String string   `formfield:"string"`
		Int    int      `formfield:"int"`
		Bool   bool     `formfield:"bool"`
		Slice  []string `formfield:"slice"`
	}{
		String: "multipart value",
		Int:    123,
		Bool:   true,
		Slice:  []string{"val1", "val2"},
	}

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("got %+v, want %+v", result, expected)
	}
}

func TestPopulate_SkippedFields(t *testing.T) {
	formData := url.Values{
		"public":  {"visible"},
		"skipped": {"should be ignored"},
		"notag":   {"also ignored"},
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var result StructWithSkippedField
	err := Populate(req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Public != "visible" {
		t.Errorf("Public: got %v, want 'visible'", result.Public)
	}
	if result.Skipped != "" {
		t.Errorf("Skipped field should be empty, got: %v", result.Skipped)
	}
	if result.NoTag != "" {
		t.Errorf("NoTag field should be empty, got: %v", result.NoTag)
	}
}

func TestPopulate_ErrorCases(t *testing.T) {
	tests := []struct {
		name        string
		target      any
		formData    url.Values
		wantErr     bool
		errContains string
	}{
		{
			name:        "non-pointer target",
			target:      BasicTypes{},
			formData:    url.Values{},
			wantErr:     true,
			errContains: "must be a pointer to a struct",
		},
		{
			name:        "non-struct target",
			target:      new(string),
			formData:    url.Values{},
			wantErr:     true,
			errContains: "must be a pointer to a struct",
		},
		{
			name: "invalid int value",
			target: &struct {
				Value int `formfield:"value"`
			}{},
			formData: url.Values{
				"value": {"not a number"},
			},
			wantErr:     true,
			errContains: "failed to set field",
		},
		{
			name: "invalid float value",
			target: &struct {
				Value float64 `formfield:"value"`
			}{},
			formData: url.Values{
				"value": {"not a float"},
			},
			wantErr:     true,
			errContains: "failed to set field",
		},
		{
			name: "invalid JSON in struct field",
			target: &struct {
				Data Contact `formfield:"data"`
			}{},
			formData: url.Values{
				"data": {`{"invalid json}`},
			},
			wantErr:     true,
			errContains: "failed to parse JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/", strings.NewReader(tt.formData.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			err := Populate(req, tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error = %v, should contain %v", err, tt.errContains)
			}
		})
	}
}

func TestPopulate_MissingFields(t *testing.T) {
	req := httptest.NewRequest("POST", "/", strings.NewReader("present=value"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var result struct {
		Present string `formfield:"present"`
		Missing string `formfield:"missing"`
	}

	err := Populate(req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Present != "value" {
		t.Errorf("Present field: got %v, want 'value'", result.Present)
	}
	if result.Missing != "" {
		t.Errorf("Missing field should remain empty, got: %v", result.Missing)
	}
}

func TestGetFile(t *testing.T) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("upload", "test.txt")
	if err != nil {
		t.Fatal(err)
	}
	content := "test file content"
	fw.Write([]byte(content))
	w.Close()

	req := httptest.NewRequest("POST", "/", &b)
	req.Header.Set("Content-Type", w.FormDataContentType())

	req.ParseMultipartForm(32 << 20)

	file, header, err := GetFile(req, "upload")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer file.Close()

	if header.Filename != "test.txt" {
		t.Errorf("filename: got %v, want 'test.txt'", header.Filename)
	}

	fileContent, _ := io.ReadAll(file)
	if string(fileContent) != content {
		t.Errorf("file content: got %v, want %v", string(fileContent), content)
	}
}

func TestPopulate_UnexportedFields(t *testing.T) {
	type StructWithUnexported struct {
		Public     string `formfield:"public"`
		unexported string `formfield:"unexported"`
	}

	req := httptest.NewRequest("POST", "/", strings.NewReader("public=public_value&unexported=should_ignore"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var result StructWithUnexported
	err := Populate(req, &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Public != "public_value" {
		t.Errorf("Public field: got %v, want 'public_value'", result.Public)
	}
	if result.unexported != "" {
		t.Errorf("Unexported field should not be set, got: %v", result.unexported)
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty slice", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result struct {
			Slice []string `formfield:"slice"`
		}
		err := Populate(req, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Slice != nil {
			t.Errorf("expected nil slice, got %v", result.Slice)
		}
	})

	t.Run("map with invalid format", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("map=invalid&map=key:value"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result struct {
			Map map[string]string `formfield:"map"`
		}
		err := Populate(req, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Map) != 1 || result.Map["key"] != "value" {
			t.Errorf("expected map with one valid entry, got %v", result.Map)
		}
	})

	t.Run("oversized array", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/", strings.NewReader("arr=1&arr=2"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result struct {
			Arr [5]int `formfield:"arr"`
		}
		err := Populate(req, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := [5]int{1, 2, 0, 0, 0}
		if result.Arr != expected {
			t.Errorf("expected %v, got %v", expected, result.Arr)
		}
	})

	t.Run("deeply nested structs", func(t *testing.T) {
		type Level3 struct {
			Value string `formfield:"value"`
		}
		type Level2 struct {
			L3 Level3 `formfield:"l3"`
		}
		type Level1 struct {
			L2 Level2 `formfield:"l2"`
		}

		req := httptest.NewRequest("POST", "/", strings.NewReader("l2.l3.value=deep"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result Level1
		err := Populate(req, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.L2.L3.Value != "deep" {
			t.Errorf("expected 'deep', got %v", result.L2.L3.Value)
		}
	})

	t.Run("nil pointer initialization", func(t *testing.T) {
		type Inner struct {
			Value string `formfield:"value"`
		}
		type Outer struct {
			Inner *Inner `formfield:"inner"`
		}

		req := httptest.NewRequest("POST", "/", strings.NewReader("inner.value=initialized"))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result Outer
		err := Populate(req, &result)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Inner == nil {
			t.Fatalf("expected Inner to be initialized")
		}
		if result.Inner.Value != "initialized" {
			t.Errorf("expected 'initialized', got %v", result.Inner.Value)
		}
	})
}

func BenchmarkPopulate_Simple(b *testing.B) {
	formData := url.Values{
		"string": {"test"},
		"int":    {"42"},
		"bool":   {"true"},
	}
	body := strings.NewReader(formData.Encode())

	for b.Loop() {
		body.Seek(0, 0)
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result BasicTypes
		Populate(req, &result)
	}
}

func BenchmarkPopulate_Complex(b *testing.B) {
	formData := url.Values{
		"slice":  {"val1", "val2", "val3", "val4", "val5"},
		"map":    {"k1:v1", "k2:v2", "k3:v3"},
		"nested": {`{"Field1":"test","Field2":123}`},
	}
	body := strings.NewReader(formData.Encode())

	for b.Loop() {
		body.Seek(0, 0)
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result struct {
			Slice  []string          `formfield:"slice"`
			Map    map[string]string `formfield:"map"`
			Nested struct {
				Field1 string `formfield:"field1"`
				Field2 int    `formfield:"field2"`
			} `formfield:"nested"`
		}
		Populate(req, &result)
	}
}

func BenchmarkPopulate_DeeplyNested(b *testing.B) {
	formData := url.Values{
		"level1.level2.level3.value": {"deep value"},
		"level1.level2.number":       {"42"},
		"level1.name":                {"top level"},
	}
	body := strings.NewReader(formData.Encode())

	type Level3 struct {
		Value string `formfield:"value"`
	}
	type Level2 struct {
		Level3 Level3 `formfield:"level3"`
		Number int    `formfield:"number"`
	}
	type Level1 struct {
		Name   string `formfield:"name"`
		Level2 Level2 `formfield:"level2"`
	}

	for b.Loop() {
		body.Seek(0, 0)
		req := httptest.NewRequest("POST", "/", body)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		var result Level1
		Populate(req, &result)
	}
}

func TestIntegrationExample(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		type LoginForm struct {
			Username  string   `formfield:"username"`
			Password  string   `formfield:"password"`
			Remember  bool     `formfield:"remember"`
			Interests []string `formfield:"interests"`
			Profile   struct {
				Age      int    `formfield:"age"`
				Location string `formfield:"location"`
			} `formfield:"profile"`
		}

		var form LoginForm
		if err := Populate(r, &form); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		response := fmt.Sprintf("Hello %s, remember=%v, interests=%v, age=%d, location=%s",
			form.Username, form.Remember, form.Interests, form.Profile.Age, form.Profile.Location)
		w.Write([]byte(response))
	}

	t.Run("flat form data", func(t *testing.T) {
		formData := url.Values{
			"username":         {"testuser"},
			"password":         {"secret"},
			"remember":         {"on"},
			"interests":        {"go", "testing", "coding"},
			"profile.age":      {"25"},
			"profile.location": {"NYC"},
		}

		req := httptest.NewRequest("POST", "/login", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		handler(rec, req)

		expected := "Hello testuser, remember=true, interests=[go testing coding], age=25, location=NYC"
		if rec.Body.String() != expected {
			t.Errorf("got %v, want %v", rec.Body.String(), expected)
		}
	})

	t.Run("JSON profile", func(t *testing.T) {
		formData := url.Values{
			"username":  {"jsonuser"},
			"password":  {"secret"},
			"remember":  {"true"},
			"interests": {"json", "api"},
			"profile":   {`{"age":30,"location":"SF"}`},
		}

		req := httptest.NewRequest("POST", "/login", strings.NewReader(formData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		rec := httptest.NewRecorder()
		handler(rec, req)

		expected := "Hello jsonuser, remember=true, interests=[json api], age=30, location=SF"
		if rec.Body.String() != expected {
			t.Errorf("got %v, want %v", rec.Body.String(), expected)
		}
	})
}

func TestPopulate_CustomTypes(t *testing.T) {
	type Status int
	const (
		StatusPending Status = iota
		StatusActive
		StatusInactive
	)

	t.Run("custom type with TextUnmarshaler", func(t *testing.T) {
		t.Skip("TextUnmarshaler support not yet implemented")
	})
}

func TestLooksLikeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`{"key":"value"}`, true},
		{`[1,2,3]`, true},
		{` {"key":"value"} `, true},
		{`{invalid}`, true},
		{`not json`, false},
		{`{`, false},
		{`}`, false},
		{`key:value`, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := looksLikeJSON(tt.input)
			if result != tt.expected {
				t.Errorf("looksLikeJSON(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

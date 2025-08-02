# Former

[![Go Reference](https://pkg.go.dev/badge/github.com/omareloui/former.svg)](https://pkg.go.dev/github.com/omareloui/former)
[![Go Report Card](https://goreportcard.com/badge/github.com/omareloui/former)](https://goreportcard.com/report/github.com/omareloui/former)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Former is a Go library for populating structs from HTTP form data using struct
field tags. It simplifies the process of binding form data to Go structures
with support for complex types, nested structures, and multipart forms.

## Features

- 🚀 **Simple API** - Just one function call to populate your structs
- 🏷️ **Tag-based binding** - Use `formfield` tags to map form fields to struct fields
- 📦 **Complex type support** - Handles slices, arrays, maps, pointers, and
  nested structs
- 🔄 **Flexible parsing** - Supports both form-urlencoded and multipart/form-data
- 🎯 **JSON support** - Parse JSON-encoded form fields into structs
- 📁 **File uploads** - Easy handling of multipart file uploads
- ⚡ **Zero dependencies** - Uses only Go standard library

## Installation

```bash
go get github.com/omareloui/former
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/omareloui/former"
)

type LoginForm struct {
    Username string   `formfield:"username"`
    Password string   `formfield:"password"`
    Remember bool     `formfield:"remember"`
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
    var form LoginForm
    if err := former.Populate(r, &form); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Use form.Username, form.Password, form.Remember
}
```

## Supported Types

### Basic Types

- `string`, `bool`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`

### Complex Types

#### Slices

Multiple values with the same field name are collected into slices:

```go
type Form struct {
    Tags []string `formfield:"tags"`
}
// Form data: tags=go&tags=web&tags=api
// Result: Tags = []string{"go", "web", "api"}
```

#### Arrays

Fixed-size arrays are filled up to their capacity:

```go
type Form struct {
    Scores [3]int `formfield:"scores"`
}
// Form data: scores=95&scores=87&scores=92
// Result: Scores = [3]int{95, 87, 92}
```

#### Maps

Maps expect `key:value` format:

```go
type Form struct {
    Settings map[string]string `formfield:"settings"`
}
// Form data: settings=theme:dark&settings=lang:en
// Result: Settings = map[string]string{"theme": "dark", "lang": "en"}
```

#### Pointers

Pointers are automatically initialized when values are present:

```go
type Form struct {
    OptionalField *string `formfield:"optional"`
}
```

## Nested Structures

### Embedded Structs

Embedded structs without tags have their fields at the top level:

```go
type Address struct {
    Street string `formfield:"street"`
    City   string `formfield:"city"`
}

type Person struct {
    Name    string `formfield:"name"`
    Address        // embedded
}
// Form data: name=John&street=Main St&city=NYC
```

### Nested with Dot Notation

Use dot notation for nested struct fields:

```go
type Order struct {
    Billing  Address `formfield:"billing"`
    Shipping Address `formfield:"shipping"`
}
// Form data: billing.street=123 Main&billing.city=NYC&shipping.street=456 Oak&shipping.city=LA
```

### JSON Support

Nested structs can be populated from JSON strings:

```go
type User struct {
    Profile Profile `formfield:"profile"`
}
// Form data: profile={"age":30,"bio":"Gopher"}
```

## Advanced Usage

### Skip Fields

Use the `-` tag to skip fields:

```go
type Form struct {
    Public  string `formfield:"public"`
    Private string `formfield:"-"`  // This field is ignored
}
```

### Custom Bool Values

Former recognizes common checkbox values:

```go
type Form struct {
    Subscribe bool `formfield:"subscribe"`
}
// Any of these values result in true: "true", "on", "1"
```

### File Uploads

Handle multipart file uploads:

```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
    // Parse form first
    var form struct {
        Title string `formfield:"title"`
    }
    if err := former.Populate(r, &form); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Get uploaded file
    file, header, err := former.GetFile(r, "document")
    if err != nil {
        http.Error(w, "File required", http.StatusBadRequest)
        return
    }
    defer file.Close()

    // Process file...
}
```

## Examples

### Complete Form Example

```go
type RegistrationForm struct {
    // Basic fields
    Username string `formfield:"username"`
    Email    string `formfield:"email"`
    Password string `formfield:"password"`
    Age      int    `formfield:"age"`

    // Multiple selections
    Interests []string `formfield:"interests"`

    // Nested struct
    Address struct {
        Street  string `formfield:"street"`
        City    string `formfield:"city"`
        Country string `formfield:"country"`
    } `formfield:"address"`

    // Map for dynamic fields
    Social map[string]string `formfield:"social"`

    // Boolean field
    Subscribe bool `formfield:"subscribe"`
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
    var form RegistrationForm
    if err := former.Populate(r, &form); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Form is populated and ready to use
}
```

### HTML Form

```html
<form method="POST" action="/register">
  <input name="username" type="text" />
  <input name="email" type="email" />
  <input name="password" type="password" />
  <input name="age" type="number" />

  <select name="interests" multiple>
    <option value="coding">Coding</option>
    <option value="music">Music</option>
    <option value="sports">Sports</option>
  </select>

  <input name="address.street" type="text" />
  <input name="address.city" type="text" />
  <input name="address.country" type="text" />

  <input name="social" value="twitter:@username" />
  <input name="social" value="github:username" />

  <input name="subscribe" type="checkbox" />

  <button type="submit">Register</button>
</form>
```

## Error Handling

Former follows these principles:

- **Missing fields**: Logged but don't cause errors
- **Type conversion errors**: Returned immediately
- **Invalid JSON**: Returns parsing error
- **Invalid target**: Must be a pointer to a struct

```go
if err := former.Populate(r, &form); err != nil {
    // Handle error - likely a type conversion issue
    log.Printf("Form parsing error: %v", err)
    http.Error(w, "Invalid form data", http.StatusBadRequest)
    return
}
```

## Performance

Former uses reflection to populate structs, which has some overhead. For best performance:

- Reuse struct types rather than creating new types dynamically
- Consider caching reflection information for frequently used types (future enhancement)
- Benchmark your specific use case if performance is critical

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE)
file for details.

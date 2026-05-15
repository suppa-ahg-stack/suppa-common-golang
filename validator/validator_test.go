package validator

import (
	"reflect"
	"testing"
)

type TestStruct struct {
	Name     string   `validation:"required,min:3,max:20"`
	Age      int      `validation:"min:0,max:120"`
	Category string   `validation:"in:foo,bar,baz"`
	Tags     []string `validation:"arrayItemSizeMin:2,arrayItemSizeMax:10"`
}

func TestValidateRequired(t *testing.T) {
	t.Run("missing required field", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Name: ""},
		}
		v.Validate()
		if !v.HasErrors() {
			t.Fatal("expected errors for missing required field")
		}
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Name" {
				if _, ok := e.InvalidConditions["required"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected required error on Name field")
		}
	})

	t.Run("required field present", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Name: "Alice"},
		}
		v.Validate()
		for _, e := range v.FieldErrors {
			if e.Field == "Name" {
				if _, ok := e.InvalidConditions["required"]; ok {
					t.Fatal("did not expect required error when field is present")
				}
			}
		}
	})
}

func TestValidateMinMax(t *testing.T) {
	t.Run("string too short", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Name: "Ab"},
		}
		v.Validate()
		if !v.HasErrors() {
			t.Fatal("expected errors")
		}
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Name" {
				if _, ok := e.InvalidConditions["min"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected min error")
		}
	})

	t.Run("string too long", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Name: "thisnameiswaytoolongtobevalid"},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Name" {
				if _, ok := e.InvalidConditions["max"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected max error")
		}
	})

	t.Run("int out of range", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Age: 200},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Age" {
				if _, ok := e.InvalidConditions["max"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected max error for Age")
		}
	})
}

func TestValidateIn(t *testing.T) {
	t.Run("value in set", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Category: "bar"},
		}
		v.Validate()
		for _, e := range v.FieldErrors {
			if e.Field == "Category" {
				t.Fatalf("unexpected error for valid category: %v", e.InvalidConditions)
			}
		}
	})

	t.Run("value not in set", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Category: "qux"},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Category" {
				if _, ok := e.InvalidConditions["in"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected in error for invalid category")
		}
	})
}

func TestValidateArrayItemSize(t *testing.T) {
	t.Run("array item too short", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Tags: []string{"a"}},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Tags" {
				if _, ok := e.InvalidConditions["arrayItemSizeMin"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected arrayItemSizeMin error")
		}
	})

	t.Run("array item too long", func(t *testing.T) {
		v := &Validator[TestStruct]{
			ToValidate: &TestStruct{Tags: []string{"supercalifragilisticexpialidocious"}},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Tags" {
				if _, ok := e.InvalidConditions["arrayItemSizeMax"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected arrayItemSizeMax error")
		}
	})
}

func TestCheckSizeMissingParam(t *testing.T) {
	v := &Validator[TestStruct]{}
	field := reflect.TypeOf(TestStruct{}).Field(0) // Name
	v.CheckSize(field, reflect.ValueOf(""), true, "", "min")
	if !v.HasErrors() {
		t.Fatal("expected error for missing param")
	}
	found := false
	for _, e := range v.FieldErrors {
		if e.Field == "Name" {
			if _, ok := e.InvalidConditions["param"]; ok {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected param error")
	}
}

func TestCheckPasswordIsValid(t *testing.T) {
	t.Run("valid password", func(t *testing.T) {
		pwd := "Hello1!"
		v := &Validator[string]{ToValidate: &pwd}
		v.CheckPasswordIsValid()
		// Note: 7 chars, should fail size (min 8)
		if !v.HasErrors() {
			t.Fatal("expected size error for 7 chars")
		}
	})

	t.Run("invalid type safe", func(t *testing.T) {
		type NotString struct{}
		ns := NotString{}
		v := &Validator[NotString]{ToValidate: &ns}
		v.CheckPasswordIsValid()
		if !v.HasErrors() {
			t.Fatal("expected type error when ToValidate is not *string")
		}
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "password" {
				if _, ok := e.InvalidConditions["type"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected type error")
		}
	})

	t.Run("missing lowercase", func(t *testing.T) {
		pwd := "HELLO1!"
		v := &Validator[string]{ToValidate: &pwd}
		v.CheckPasswordIsValid()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "password" {
				if _, ok := e.InvalidConditions["minChar"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected minChar error for missing lowercase")
		}
	})

	t.Run("missing uppercase", func(t *testing.T) {
		pwd := "hello1!"
		v := &Validator[string]{ToValidate: &pwd}
		v.CheckPasswordIsValid()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "password" {
				if _, ok := e.InvalidConditions["capChar"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected capChar error for missing uppercase")
		}
	})

	t.Run("missing number", func(t *testing.T) {
		pwd := "Hello!!"
		v := &Validator[string]{ToValidate: &pwd}
		v.CheckPasswordIsValid()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "password" {
				if _, ok := e.InvalidConditions["number"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected number error for missing digit")
		}
	})

	t.Run("missing special char", func(t *testing.T) {
		pwd := "Hello11"
		v := &Validator[string]{ToValidate: &pwd}
		v.CheckPasswordIsValid()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "password" {
				if _, ok := e.InvalidConditions["specialChar"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected specialChar error for missing special char")
		}
	})
}

func TestValidateRegex(t *testing.T) {
	t.Run("match regex", func(t *testing.T) {
		type RegexStruct struct {
			Email string `validation:"regex:^[a-z]+@[a-z]+\\.[a-z]+$"`
		}
		v := &Validator[RegexStruct]{
			ToValidate: &RegexStruct{Email: "test@example.com"},
		}
		v.Validate()
		for _, e := range v.FieldErrors {
			if e.Field == "Email" {
				t.Fatalf("unexpected error for valid regex: %v", e.InvalidConditions)
			}
		}
	})

	t.Run("no match regex", func(t *testing.T) {
		type RegexStruct struct {
			Email string `validation:"regex:^[a-z]+@[a-z]+\\.[a-z]+$"`
		}
		v := &Validator[RegexStruct]{
			ToValidate: &RegexStruct{Email: "invalid-email"},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Email" {
				if _, ok := e.InvalidConditions["regex"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected regex error for non-matching value")
		}
	})

	t.Run("invalid regex pattern", func(t *testing.T) {
		type RegexStruct struct {
			Email string `validation:"regex:[invalid"`
		}
		v := &Validator[RegexStruct]{
			ToValidate: &RegexStruct{Email: "test@example.com"},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Email" {
				if _, ok := e.InvalidConditions["param"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected param error for invalid regex pattern")
		}
	})

	t.Run("regex on non-string field", func(t *testing.T) {
		type RegexStruct struct {
			Age int `validation:"regex:^\\d+$"`
		}
		v := &Validator[RegexStruct]{
			ToValidate: &RegexStruct{Age: 25},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Age" {
				if _, ok := e.InvalidConditions["regex"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected regex error for non-string field")
		}
	})

	t.Run("missing regex param", func(t *testing.T) {
		type RegexStruct struct {
			Email string `validation:"regex:"`
		}
		v := &Validator[RegexStruct]{
			ToValidate: &RegexStruct{Email: "test"},
		}
		v.Validate()
		found := false
		for _, e := range v.FieldErrors {
			if e.Field == "Email" {
				if _, ok := e.InvalidConditions["param"]; ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatal("expected param error for empty regex pattern")
		}
	})
}

func TestStringMustContain(t *testing.T) {
	if !StringMustContain("hello", "aeiou", 1) {
		t.Fatal("expected StringMustContain to find a vowel")
	}
	if StringMustContain("xyz", "aeiou", 1) {
		t.Fatal("expected StringMustContain to fail without vowels")
	}
}

func TestStringSizeBetween(t *testing.T) {
	if !StringSizeBetween("hello", 3, 10) {
		t.Fatal("expected StringSizeBetween to pass")
	}
	if StringSizeBetween("hi", 3, 10) {
		t.Fatal("expected StringSizeBetween to fail (too short)")
	}
	if StringSizeBetween("this is way too long", 3, 10) {
		t.Fatal("expected StringSizeBetween to fail (too long)")
	}
}

package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type ValidationError struct {
	Field             string
	InvalidConditions map[string]string
}

type PasswordRules struct {
	min                  int
	max                  int
	numberOfSpecialChars int
	numberOfUpperChars   int
	numberOfLowerChars   int
	numberOfDigits       int
}

var keyValidations = []string{
	"min", "max", "arrayItemSizeMin", "arrayItemSizeMax", "required", "in", "regex",
}

// Define a new Validator struct which contains a map of validation error messages
// for our form fields.
type Validator[T any] struct {
	FieldErrors []ValidationError
	ToValidate  *T
}

func (v *Validator[T]) Validate() {
	fields := reflect.TypeOf(*v.ToValidate)
	values := reflect.ValueOf(*v.ToValidate)
	for i := 0; i < fields.NumField(); i++ {
		field := fields.Field(i)
		if validationString, ok := field.Tag.Lookup("validation"); ok {
			// les règles sont séparées par une virgule
			rules := strings.Split(validationString, ",")
			if len(rules) > 0 {
				for rIdx := 0; rIdx < len(rules); rIdx++ {
					rule := rules[rIdx]
					// le nom de la règle et le paramêtre, s'il y en a, est séparée par un :
					splitted := strings.SplitN(rule, ":", 2)
					key := splitted[0]
					var param string
					if len(splitted) > 1 {
						param = splitted[1]
						// Ré-assembler les règles suivantes sans ':' comme faisant partie du paramètre actuel
						for rIdx+1 < len(rules) && !strings.Contains(rules[rIdx+1], ":") {
							rIdx++
							param += "," + rules[rIdx]
						}
					}
					if ok := StringInStringArray(key, keyValidations); ok {
						if key == "required" {
							v.CheckRequired(field, values.Field(i), key)
						} else if !values.Field(i).IsZero() {
							switch key {
							case "min":
								v.CheckSize(field, values.Field(i), true, param, key)
							case "max":
								v.CheckSize(field, values.Field(i), false, param, key)
							case "arrayItemSizeMin":
								length := values.Field(i).Len()
								for j := range length {
									v.CheckSize(field, values.Field(i).Index(j), true, param, key)
								}
							case "arrayItemSizeMax":
								length := values.Field(i).Len()
								for j := range length {
									v.CheckSize(field, values.Field(i).Index(j), false, param, key)
								}
							case "in":
								v.CheckIn(field, values.Field(i), param, key)
							case "regex":
								v.CheckRegex(field, values.Field(i), param, key)
							}
						}
					}
				}
			}
		}
	}
}

func (v *Validator[T]) CheckSize(field reflect.StructField, value reflect.Value, forMin bool, param string, checkFor string) {
	fieldKind := field.Type.Kind()
	if param == "" {
		v.AddError(field.Name, "param", "le parametre est manquant")
		return
	}
	paramValue, err := strconv.Atoi(param)
	if err != nil {
		v.AddError(field.Name, "param", fmt.Sprintf("le parametre n'a pas pu être converti en int avec error : %s", err.Error()))
		return
	}
	if forMin && paramValue < 0 {
		v.AddError(field.Name, "param", fmt.Sprintf("le parametre ne peut pas être négatif avec params %d", paramValue))
		return
	}

	hasErrorMin := false
	hasErrorMax := false
	if fieldKind == reflect.String {
		if forMin && len(value.String()) < paramValue {
			hasErrorMin = true
		} else if !forMin && len(value.String()) > paramValue {
			hasErrorMax = true
		}
	} else if fieldKind == reflect.Int {
		intVal := int(value.Int())
		if forMin && intVal < paramValue {
			hasErrorMin = true
		} else if !forMin && intVal > paramValue {
			hasErrorMax = true
		}
	} else if fieldKind == reflect.Array || fieldKind == reflect.Slice {
		if forMin && value.Len() < paramValue {
			hasErrorMin = true
		} else if !forMin && value.Len() > paramValue {
			hasErrorMax = true
		}
	}

	if hasErrorMin {
		v.AddError(field.Name, checkFor, fmt.Sprintf("taille insuffisante pour %s = %v", checkFor, paramValue))
	} else if hasErrorMax {
		v.AddError(field.Name, checkFor, fmt.Sprintf("taille dépassée pour %s = %v", checkFor, paramValue))
	}
}

func (v *Validator[T]) HasErrors() bool {
	return len(v.FieldErrors) > 0
}

func (v *Validator[T]) AddError(field string, item string, message string) {
	for idx, fieldError := range v.FieldErrors {
		if fieldError.Field == field {
			v.FieldErrors[idx].InvalidConditions[item] = message
			return
		}
	}

	fieldError := ValidationError{
		Field: field,
	}
	invalidConditions := make(map[string]string)
	invalidConditions[item] = message
	fieldError.InvalidConditions = invalidConditions
	v.FieldErrors = append(v.FieldErrors, fieldError)
}

func (v *Validator[T]) CheckIn(field reflect.StructField, value reflect.Value, param string, checkFor string) {
	inValues := strings.Split(param, ",")

	found := false
	for i := range inValues {
		if inValues[i] == value.String() {
			found = true
			break
		}
	}

	if !found {
		v.AddError(field.Name, "in", fmt.Sprintf("élément manquant pour %s", checkFor))
	}
}

func (v *Validator[T]) CheckRequired(field reflect.StructField, value reflect.Value, checkFor string) {
	if value.IsZero() {
		v.AddError(field.Name, checkFor, fmt.Sprintf("élément manquant pour %s", checkFor))
	}
}

func (v *Validator[T]) CheckRegex(field reflect.StructField, value reflect.Value, param string, checkFor string) {
	if param == "" {
		v.AddError(field.Name, "param", "le pattern regex est manquant")
		return
	}
	if field.Type.Kind() != reflect.String {
		v.AddError(field.Name, "regex", "la validation regex ne s'applique qu'aux chaînes de caractères")
		return
	}
	re, err := regexp.Compile(param)
	if err != nil {
		v.AddError(field.Name, "param", fmt.Sprintf("le pattern regex est invalide : %s", err.Error()))
		return
	}
	if !re.MatchString(value.String()) {
		v.AddError(field.Name, "regex", fmt.Sprintf("la valeur ne correspond pas au pattern %s", checkFor))
	}
}

func (v *Validator[T]) CheckPasswordIsValid(rules PasswordRules) {
	password, ok := any(v.ToValidate).(*string)
	if !ok {
		v.AddError("password", "type", "invalid_type")
		return
	}

	if len((*password)) < 8 {
		v.AddError("password", "size", fmt.Sprintf("invalid_min_chars_size__%d", rules.min))
	}

	if len((*password)) > 32 {
		v.AddError("password", "size", fmt.Sprintf("invalid_max_chars_size__%d", rules.max))
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString((*password))
	hasLower := regexp.MustCompile(`[a-z]`).MatchString((*password))
	hasDigit := regexp.MustCompile(`\d`).MatchString((*password))
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_\-+=\[\]{};:'",.<>/?\\|` + "`" + `~]`).MatchString((*password))
	validChars := regexp.MustCompile(`^[A-Za-z\d!@#$%^&*()_\-+=\[\]{};:'",.<>/?\\|` + "`" + `~]+$`).MatchString((*password))

	if !hasUpper {
		v.AddError("password", "upperChar", fmt.Sprintf("capital_letters_required__%d", rules.numberOfUpperChars))
	}

	if !hasLower {
		v.AddError("password", "lowerChar", fmt.Sprintf("lower_letters_required__%d", rules.numberOfLowerChars))
	}

	if !hasDigit {
		v.AddError("password", "digit", fmt.Sprintf("digit_required__%d", rules.numberOfDigits))
	}

	if !hasSpecial {
		v.AddError("password", "specialChar", fmt.Sprintf("special_chars_required__%d", rules.numberOfSpecialChars))
	}

	if !validChars {
		v.AddError("password", "invalidChar", "valid_chars_required")
	}
}

func (v *Validator[T]) CheckEmailIsValid() {
	email, ok := any(v.ToValidate).(*string)
	if !ok {
		v.AddError("email", "type", "invalid_type")
		return
	}
	isValid := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`).MatchString((*email))
	if !isValid {
		v.AddError("email", "invalid", "email_invalid")
	}
}

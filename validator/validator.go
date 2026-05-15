package validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var (
	specialChars string = "@.#$!%*?&^()/+-"
	minChars     string = "abcdefghijklmnopqrstuvwxyz"
	capChars     string = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	numbers      string = "0123456789"
)

type ValidationError struct {
	Field             string
	InvalidConditions map[string]string
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

func (v *Validator[T]) CheckPasswordIsValid() {
	password, ok := any(v.ToValidate).(*string)
	if !ok {
		v.AddError("password", "type", "le type de données n'est pas valide pour la vérification du mot de passe")
		return
	}
	res := StringSizeBetween(*password, 8, 32)
	if !res {
		v.AddError("password", "size", "Le mot de passe doit être entre 8 et 32 charactères")
	}
	res = StringMustContain(*password, specialChars, 1)
	if !res {
		v.AddError("password", "specialChar", "Le mot de passe doit contenir au moins un charactère spécial parmi "+specialChars)
	}
	res = StringMustContain(*password, capChars, 1)
	if !res {
		v.AddError("password", "capChar", "Le mot de passe doit contenir au moins une majuscule")
	}
	res = StringMustContain(*password, minChars, 1)
	if !res {
		v.AddError("password", "minChar", "Le mot de passe doit contenir au moins une minuscule")
	}
	res = StringMustContain(*password, numbers, 1)
	if !res {
		v.AddError("password", "number", "Le mot de passe doit contenir au moins un chiffre")
	}
}

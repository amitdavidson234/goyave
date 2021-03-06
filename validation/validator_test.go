package validation

import (
	"reflect"
	"testing"

	"github.com/System-Glitch/goyave/v2/lang"
	"github.com/stretchr/testify/suite"
)

type ValidatorTestSuite struct {
	suite.Suite
}

func (suite *ValidatorTestSuite) SetupSuite() {
	lang.LoadDefault()
}

func (suite *ValidatorTestSuite) TestIsTypeDependant() {
	suite.True(isTypeDependent("min"))
	suite.False(isTypeDependent("required"))
}

func (suite *ValidatorTestSuite) TestIsRequired() {
	suite.True(isRequired([]string{"string", "required", "min:5"}))
	suite.False(isRequired([]string{"string", "min:5"}))
}

func (suite *ValidatorTestSuite) TestIsNullable() {
	suite.True(isNullable([]string{"string", "required", "nullable", "min:5"}))
	suite.False(isNullable([]string{"string", "min:5", "required"}))
}

func (suite *ValidatorTestSuite) TestIsArray() {
	suite.True(isArray([]string{"array", "required", "nullable", "min:5"}))
	suite.False(isArray([]string{"string", "min:5", "required"}))
}

func (suite *ValidatorTestSuite) TestArrayType() {
	suite.True(isArrayType("integer"))
	suite.False(isArrayType("file"))
}

func (suite *ValidatorTestSuite) TestParseRule() {
	rule, _, params := parseRule("required")
	suite.Equal("required", rule)
	suite.Equal(0, len(params))

	rule, _, params = parseRule("min:5")
	suite.Equal("min", rule)
	suite.Equal(1, len(params))
	suite.Equal("5", params[0])

	suite.Panics(func() {
		parseRule("invalid,rule")
	})

	suite.Panics(func() {
		parseRule("invalidrule")
	})

	rule, validatesArray, params := parseRule(">min:3")
	suite.Equal("min", rule)
	suite.Equal(1, len(params))
	suite.Equal("3", params[0])
	suite.Equal(uint8(1), validatesArray)

	suite.Panics(func() {
		parseRule(">file")
	})

	rule, validatesArray, params = parseRule(">>max:5")
	suite.Equal("max", rule)
	suite.Equal(1, len(params))
	suite.Equal("5", params[0])
	suite.Equal(uint8(2), validatesArray)
}

func (suite *ValidatorTestSuite) TestGetMessage() {
	suite.Equal("The :field is required.", getMessage("required", reflect.ValueOf("test"), "en-US", 0))
	suite.Equal("The :field must be at least :min.", getMessage("min", reflect.ValueOf(42), "en-US", 0))
	suite.Equal("The :field values must be at least :min.", getMessage("min", reflect.ValueOf(42), "en-US", 1))
}

func (suite *ValidatorTestSuite) TestAddRule() {
	suite.Panics(func() {
		AddRule("required", false, func(field string, value interface{}, parameters []string, form map[string]interface{}) bool {
			return false
		})
	})

	AddRule("new_rule", false, func(field string, value interface{}, parameters []string, form map[string]interface{}) bool {
		return true
	})
	_, ok := validationRules["new_rule"]
	suite.True(ok)
	suite.False(isTypeDependent("new_rule"))

	AddRule("new_rule_type_dependent", true, func(field string, value interface{}, parameters []string, form map[string]interface{}) bool {
		return true
	})
	_, ok = validationRules["new_rule_type_dependent"]
	suite.True(ok)
	suite.True(isTypeDependent("new_rule_type_dependent"))
}

func (suite *ValidatorTestSuite) TestValidate() {
	errors := Validate(nil, RuleSet{}, false, "en-US")
	suite.Equal(1, len(errors))
	suite.Equal("Malformed request", errors["error"][0])

	errors = Validate(nil, RuleSet{}, true, "en-US")
	suite.Equal(1, len(errors))
	suite.Equal("Malformed JSON", errors["error"][0])

	errors = Validate(map[string]interface{}{
		"string": "hello world",
		"number": 42,
	}, RuleSet{
		"string": {"required", "string"},
		"number": {"required", "numeric", "min:10"},
	}, true, "en-US")
	suite.Equal(0, len(errors))

	data := map[string]interface{}{
		"nullField": nil,
	}
	errors = Validate(data, RuleSet{
		"nullField": {"numeric"},
	}, true, "en-US")
	_, exists := data["nullField"]
	suite.False(exists)
	suite.Equal(0, len(errors))

	data = map[string]interface{}{
		"nullField": nil,
	}
	errors = Validate(data, RuleSet{
		"nullField": {"required", "nullable", "numeric"},
	}, true, "en-US")
	_, exists = data["nullField"]
	suite.True(exists)
	suite.Equal(0, len(errors))

	data = map[string]interface{}{
		"nullField": "test",
	}
	errors = Validate(data, RuleSet{
		"nullField": {"required", "nullable", "numeric"},
	}, true, "en-US")
	_, exists = data["nullField"]
	suite.True(exists)
	suite.Equal(1, len(errors))
}

func (suite *ValidatorTestSuite) TestValidateWithArray() {
	data := map[string]interface{}{
		"string": "hello",
	}
	errors := Validate(data, RuleSet{
		"string": {"required", "array"},
	}, false, "en-US")
	suite.Equal("array", GetFieldType(data["string"]))
	suite.Equal("hello", data["string"].([]string)[0])
	suite.Equal(0, len(errors))
}

func (suite *ValidatorTestSuite) TestValidateArrayValues() {
	data := map[string]interface{}{
		"string": []string{"hello", "world"},
	}
	errors := Validate(data, RuleSet{
		"string": {"required", "array", ">min:3"},
	}, false, "en-US")
	suite.Equal(0, len(errors))

	data = map[string]interface{}{
		"string": []string{"hi", ",", "there"},
	}
	errors = Validate(data, RuleSet{
		"string": {"required", "array", ">min:3"},
	}, false, "en-US")
	suite.Equal(1, len(errors))

	data = map[string]interface{}{
		"string": []string{"johndoe@example.org", "foobar@example.org"},
	}
	errors = Validate(data, RuleSet{
		"string": {"required", "array:string", ">email"},
	}, true, "en-US")
	suite.Equal(0, len(errors))

	suite.Panics(func() {
		validateRuleInArray("required", "string", 1, map[string]interface{}{"string": "hi"}, []string{})
	})

	// Empty array
	data = map[string]interface{}{
		"string": []string{},
	}
	errors = Validate(data, RuleSet{
		"string": {"array", ">uuid:5"},
	}, true, "en-US")
	suite.Equal(0, len(errors))
}

func (suite *ValidatorTestSuite) TestValidateTwoDimensionalArray() {
	data := map[string]interface{}{
		"values": [][]interface{}{{"0.5", 1.42}, {0.6, 7}},
	}
	errors := Validate(data, RuleSet{
		"values": {"required", "array", ">array:numeric"},
	}, false, "en-US")
	suite.Equal(0, len(errors))

	arr, ok := data["values"].([][]float64)
	suite.True(ok)
	if ok {
		suite.Equal(2, len(arr))
		suite.Equal(0.5, arr[0][0])
		suite.Equal(1.42, arr[0][1])
		suite.Equal(0.6, arr[1][0])
		suite.Equal(7.0, arr[1][1])
	}

	data = map[string]interface{}{
		"values": [][]float64{{5, 8}, {0.6, 7}},
	}
	errors = Validate(data, RuleSet{
		"values": {"required", "array", ">array:numeric", ">min:3"},
	}, true, "en-US")
	suite.Equal(1, len(errors))

	_, ok = data["values"].([][]float64)
	suite.True(ok)

	data = map[string]interface{}{
		"values": [][]float64{{5, 8, 6}, {0.6, 7, 9}},
	}
	errors = Validate(data, RuleSet{
		"values": {"required", "array", ">array:numeric", ">min:3"},
	}, true, "en-US")
	suite.Equal(0, len(errors))

	data = map[string]interface{}{
		"values": [][]float64{{5, 8}, {3, 7}},
	}
	errors = Validate(data, RuleSet{
		"values": {"required", "array", ">array:numeric", ">>min:3"},
	}, true, "en-US")
	suite.Equal(0, len(errors))

	data = map[string]interface{}{
		"values": [][]float64{{5, 8}, {0.6, 7}},
	}
	errors = Validate(data, RuleSet{
		"values": {"required", "array", ">array:numeric", ">>min:3"},
	}, true, "en-US")
	suite.Equal(1, len(errors))
}

func (suite *ValidatorTestSuite) TestValidateNDimensionalArray() {
	data := map[string]interface{}{
		"values": [][][]interface{}{
			{{"0.5", 1.42}, {0.6, 4, 3}},
			{{"0.6", "1.43"}, {}, {2}},
		},
	}
	errors := Validate(data, RuleSet{
		"values": {"required", "array", ">array", ">>array:numeric", ">max:3", ">>>max:4"},
	}, true, "en-US")
	suite.Equal(0, len(errors))

	arr, ok := data["values"].([][][]float64)
	suite.True(ok)
	if ok {
		suite.Equal(2, len(arr))
		suite.Equal(2, len(arr[0]))
		suite.Equal(3, len(arr[1]))
		suite.Equal(0.5, arr[0][0][0])
		suite.Equal(1.42, arr[0][0][1])
		suite.Equal(2.0, arr[1][2][0])
	}

	data = map[string]interface{}{
		"values": [][][]interface{}{
			{{"0.5", 1.42}, {0.6, 4, 3}},
			{{"0.6", "1.43"}, {}, {2}, {4}},
		},
	}
	errors = Validate(data, RuleSet{
		"values": {"required", "array", ">array", ">>array:numeric", ">max:3", ">>>max:4"},
	}, true, "en-US")
	suite.Equal(1, len(errors))

	data = map[string]interface{}{
		"values": [][][]interface{}{
			{{"0.5", 1.42}, {0.6, 9, 3}},
			{{"0.6", "1.43"}, {}, {2}},
		},
	}
	errors = Validate(data, RuleSet{
		"values": {"required", "array", ">array", ">>array:numeric", ">max:3", ">>>max:4"},
	}, true, "en-US")
	suite.Equal(1, len(errors))
}

func TestValidatorTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatorTestSuite))
}

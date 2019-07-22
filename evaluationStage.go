package govaluate

import (
	"errors"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strings"
)

const (
	logicalErrorFormat    string = "Value '%v' cannot be used with the logical operator '%v', it is not a bool"
	modifierErrorFormat   string = "Value '%v' cannot be used with the modifier '%v', it is not a number"
	comparatorErrorFormat string = "Value '%v' cannot be used with the comparator '%v', it is not a number"
	ternaryErrorFormat    string = "Value '%v' cannot be used with the ternary operator '%v', it is not a bool"
	prefixErrorFormat     string = "Value '%v' cannot be used with the prefix '%v'"
)

type evaluationOperator func(left interface{}, right interface{}, parameters Parameters) (interface{}, error)
type stageTypeCheck func(value interface{}) bool
type stageCombinedTypeCheck func(left interface{}, right interface{}) bool

type evaluationStage struct {
	symbol OperatorSymbol

	leftStage, rightStage *evaluationStage

	// the operation that will be used to evaluate this stage (such as adding [left] to [right] and return the result)
	operator evaluationOperator

	// ensures that both left and right values are appropriate for this stage. Returns an error if they aren't operable.
	leftTypeCheck  stageTypeCheck
	rightTypeCheck stageTypeCheck

	// if specified, will override whatever is used in "leftTypeCheck" and "rightTypeCheck".
	// primarily used for specific operators that don't care which side a given type is on, but still requires one side to be of a given type
	// (like string concat)
	typeCheck stageCombinedTypeCheck

	// regardless of which type check is used, this string format will be used as the error message for type errors
	typeErrorFormat string
}

var (
	_true  = interface{}(true)
	_false = interface{}(false)
)

func (this *evaluationStage) swapWith(other *evaluationStage) {

	temp := *other
	other.setToNonStage(*this)
	this.setToNonStage(temp)
}

func (this *evaluationStage) setToNonStage(other evaluationStage) {

	this.symbol = other.symbol
	this.operator = other.operator
	this.leftTypeCheck = other.leftTypeCheck
	this.rightTypeCheck = other.rightTypeCheck
	this.typeCheck = other.typeCheck
	this.typeErrorFormat = other.typeErrorFormat
}

func (this *evaluationStage) isShortCircuitable() bool {

	switch this.symbol {
	case AND:
		fallthrough
	case OR:
		fallthrough
	case TERNARY_TRUE:
		fallthrough
	case TERNARY_FALSE:
		fallthrough
	case COALESCE:
		return true
	}

	return false
}

func getNoData(parameters Parameters) (float32, error) {
	noData := float32(math.SmallestNonzeroFloat32)
	val, err := parameters.Get("nodata")
	if err == nil {
		v, ok := val.(float32)
		if !ok {
			return 0, fmt.Errorf("invalid nodata value: %v", val)
		}
		noData = v
	}

	return noData, nil
}

func noopStageRight(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	return right, nil
}

func addStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	// string concat if either are strings
	if isString(left) || isString(right) {
		return fmt.Sprintf("%v%v", left, right), nil
	}

	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] + rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] + rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = lx + rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx + rx, nil
	}

	return nil, fmt.Errorf("invalid operand for addition")

}
func subtractStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] - rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] - rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = lx - rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx - rx, nil
	}

	return nil, fmt.Errorf("invalid operand for subtraction")
}
func multiplyStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] * rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] * rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = lx * rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx * rx, nil
	}

	return nil, fmt.Errorf("invalid operand for multiplication")
}
func divideStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] / rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = lax[i] / rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = lx / rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx / rx, nil
	}

	return nil, fmt.Errorf("invalid operand for division")
}
func exponentStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(math.Pow(float64(lax[i]), float64(rax[i])))
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(math.Pow(float64(lax[i]), float64(rx)))
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(math.Pow(float64(lx), float64(rax[i])))
		}
		return res, nil
	}

	if lok && rok {
		return float32(math.Pow(float64(lx), float64(rx))), nil
	}

	return nil, fmt.Errorf("invalid operand for exponential")

}
func modulusStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(math.Mod(float64(lax[i]), float64(rax[i])))
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(math.Mod(float64(lax[i]), float64(rx)))
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(math.Mod(float64(lx), float64(rax[i])))
		}
		return res, nil
	}

	if lok && rok {
		return float32(math.Mod(float64(lx), float64(rx))), nil
	}

	return nil, fmt.Errorf("invalid operand for modulus")
}
func gteStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) >= right.(string)), nil
	}

	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] >= rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] >= rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx >= rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx >= rx, nil
	}

	return nil, fmt.Errorf("invalid operand for >=")
}
func gtStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) > right.(string)), nil
	}

	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] > rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] > rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx > rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx > rx, nil
	}

	return nil, fmt.Errorf("invalid operand for >")
}
func lteStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) <= right.(string)), nil
	}

	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] <= rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] <= rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx <= rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx <= rx, nil
	}

	return nil, fmt.Errorf("invalid operand for <=")
}
func ltStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	if isString(left) && isString(right) {
		return boolIface(left.(string) < right.(string)), nil
	}
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] < rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] < rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx < rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx < rx, nil
	}

	return nil, fmt.Errorf("invalid operand for <")
}
func equalStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] == rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] == rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx == rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx == rx, nil
	}

	return nil, fmt.Errorf("invalid operand for ==")
}
func notEqualStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] != rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] != rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx != rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx != rx, nil
	}

	return nil, fmt.Errorf("invalid operand for !=")
}
func andStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]bool)
	lx, lok := left.(bool)

	rax, raok := right.([]bool)
	rx, rok := right.(bool)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] && rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] && rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx && rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx && rx, nil
	}

	return nil, fmt.Errorf("invalid operand for &&")

	return boolIface(left.(bool) && right.(bool)), nil
}
func orStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]bool)
	lx, lok := left.(bool)

	rax, raok := right.([]bool)
	rx, rok := right.(bool)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] || rax[i]
		}
		return res, nil
	}

	if laok && rok {
		res := make([]bool, len(lax))
		for i := range lax {
			res[i] = lax[i] || rx
		}
		return res, nil
	}

	if lok && raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = lx || rax[i]
		}
		return res, nil
	}

	if lok && rok {
		return lx || rx, nil
	}

	return nil, fmt.Errorf("invalid operand for ||")

}
func negateStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = -rax[i]
		}
		return res, nil
	}

	if rok {
		return -rx, nil
	}

	return nil, fmt.Errorf("invalid operand for -")
}
func invertStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	rax, raok := right.([]bool)
	rx, rok := right.(bool)

	if raok {
		res := make([]bool, len(rax))
		for i := range rax {
			res[i] = !rax[i]
		}
		return res, nil
	}

	if rok {
		return !rx, nil
	}

	return nil, fmt.Errorf("invalid operand for !")
}
func bitwiseNotStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(^int64(rax[i]))
		}
		return res, nil
	}

	if rok {
		return float32(^int64(rx)), nil
	}

	return nil, fmt.Errorf("invalid operand for ^")
}
func ternaryIfStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	noData, err := getNoData(parameters)
	if err != nil {
		return nil, err
	}

	lax, laok := left.([]bool)
	lx, lok := left.(bool)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			if lax[i] {
				res[i] = rax[i]
			} else {
				res[i] = noData
			}
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			if lax[i] {
				res[i] = rx
			} else {
				res[i] = noData
			}
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			if lx {
				res[i] = rax[i]
			} else {
				res[i] = noData
			}
		}
		return res, nil
	}

	if lok && rok {
		if lx {
			return rx, nil
		} else {
			return noData, nil
		}
	}

	return nil, fmt.Errorf("invalid operand for ternary if")
}
func ternaryElseStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	noData, err := getNoData(parameters)
	if err != nil {
		return nil, err
	}

	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			if lax[i] == noData {
				res[i] = rax[i]
			} else {
				res[i] = lax[i]
			}
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			if lax[i] == noData {
				res[i] = rx
			} else {
				res[i] = lax[i]
			}
		}
		return res, nil
	}

	if (lok || lx == noData) && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			if lx == noData {
				res[i] = rax[i]
			} else {
				res[i] = lx
			}
		}
		return res, nil
	}

	if (lok || lx == noData) && rok {
		if lx == noData {
			return rx, nil
		} else {
			return lx, nil
		}
	}

	return nil, fmt.Errorf("invalid operand for ternary else")
}

func regexStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	var pattern *regexp.Regexp
	var err error

	switch right.(type) {
	case string:
		pattern, err = regexp.Compile(right.(string))
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Unable to compile regexp pattern '%v': %v", right, err))
		}
	case *regexp.Regexp:
		pattern = right.(*regexp.Regexp)
	}

	return pattern.Match([]byte(left.(string))), nil
}

func notRegexStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	ret, err := regexStage(left, right, parameters)
	if err != nil {
		return nil, err
	}

	return !(ret.(bool)), nil
}

func bitwiseOrStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(int64(lax[i]) | int64(rax[i]))
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(int64(lax[i]) | int64(rx))
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(int64(lx) | int64(rax[i]))
		}
		return res, nil
	}

	if lok && rok {
		return float32(int64(lx) | int64(rx)), nil
	}

	return nil, fmt.Errorf("invalid operand for |")

}
func bitwiseAndStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(int64(lax[i]) & int64(rax[i]))
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(int64(lax[i]) & int64(rx))
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(int64(lx) & int64(rax[i]))
		}
		return res, nil
	}

	if lok && rok {
		return float32(int64(lx) & int64(rx)), nil
	}

	return nil, fmt.Errorf("invalid operand for &")
}
func bitwiseXORStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(int64(lax[i]) ^ int64(rax[i]))
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(int64(lax[i]) ^ int64(rx))
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(int64(lx) ^ int64(rax[i]))
		}
		return res, nil
	}

	if lok && rok {
		return float32(int64(lx) ^ int64(rx)), nil
	}

	return nil, fmt.Errorf("invalid operand for ^")
}
func leftShiftStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(uint64(lax[i]) << uint64(rax[i]))
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(uint64(lax[i]) << uint64(rx))
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(uint64(lx) << uint64(rax[i]))
		}
		return res, nil
	}

	if lok && rok {
		return float32(uint64(lx) << uint64(rx)), nil
	}

	return nil, fmt.Errorf("invalid operand for <<")
}
func rightShiftStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
	lax, laok := left.([]float32)
	lx, lok := left.(float32)

	rax, raok := right.([]float32)
	rx, rok := right.(float32)

	if laok && raok {
		if len(lax) != len(rax) {
			return nil, fmt.Errorf("different array sizes: %v, %v", len(lax), len(rax))
		}

		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(uint64(lax[i]) >> uint64(rax[i]))
		}
		return res, nil
	}

	if laok && rok {
		res := make([]float32, len(lax))
		for i := range lax {
			res[i] = float32(uint64(lax[i]) >> uint64(rx))
		}
		return res, nil
	}

	if lok && raok {
		res := make([]float32, len(rax))
		for i := range rax {
			res[i] = float32(uint64(lx) >> uint64(rax[i]))
		}
		return res, nil
	}

	if lok && rok {
		return float32(uint64(lx) >> uint64(rx)), nil
	}

	return nil, fmt.Errorf("invalid operand for >>")
}

func makeParameterStage(parameterName string) evaluationOperator {

	return func(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
		value, err := parameters.Get(parameterName)
		if err != nil {
			return nil, err
		}

		return value, nil
	}
}

func makeLiteralStage(literal interface{}) evaluationOperator {
	return func(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {
		return literal, nil
	}
}

func makeFunctionStage(function ExpressionFunction) evaluationOperator {

	return func(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

		if right == nil {
			return function()
		}

		switch right.(type) {
		case []interface{}:
			return function(right.([]interface{})...)
		default:
			return function(right)
		}
	}
}

func typeConvertParam(p reflect.Value, t reflect.Type) (ret reflect.Value, err error) {
	defer func() {
		if r := recover(); r != nil {
			errorMsg := fmt.Sprintf("Argument type conversion failed: failed to convert '%s' to '%s'", p.Kind().String(), t.Kind().String())
			err = errors.New(errorMsg)
			ret = p
		}
	}()

	return p.Convert(t), nil
}

func typeConvertParams(method reflect.Value, params []reflect.Value) ([]reflect.Value, error) {

	methodType := method.Type()
	numIn := methodType.NumIn()
	numParams := len(params)

	if numIn != numParams {
		if numIn > numParams {
			return nil, fmt.Errorf("Too few arguments to parameter call: got %d arguments, expected %d", len(params), numIn)
		}
		return nil, fmt.Errorf("Too many arguments to parameter call: got %d arguments, expected %d", len(params), numIn)
	}

	for i := 0; i < numIn; i++ {
		t := methodType.In(i)
		p := params[i]
		pt := p.Type()

		if t.Kind() != pt.Kind() {
			np, err := typeConvertParam(p, t)
			if err != nil {
				return nil, err
			}
			params[i] = np
		}
	}

	return params, nil
}

func makeAccessorStage(pair []string) evaluationOperator {

	reconstructed := strings.Join(pair, ".")

	return func(left interface{}, right interface{}, parameters Parameters) (ret interface{}, err error) {

		var params []reflect.Value

		value, err := parameters.Get(pair[0])
		if err != nil {
			return nil, err
		}

		// while this library generally tries to handle panic-inducing cases on its own,
		// accessors are a sticky case which have a lot of possible ways to fail.
		// therefore every call to an accessor sets up a defer that tries to recover from panics, converting them to errors.
		defer func() {
			if r := recover(); r != nil {
				errorMsg := fmt.Sprintf("Failed to access '%s': %v", reconstructed, r.(string))
				err = errors.New(errorMsg)
				ret = nil
			}
		}()

		for i := 1; i < len(pair); i++ {

			coreValue := reflect.ValueOf(value)

			var corePtrVal reflect.Value

			// if this is a pointer, resolve it.
			if coreValue.Kind() == reflect.Ptr {
				corePtrVal = coreValue
				coreValue = coreValue.Elem()
			}

			if coreValue.Kind() != reflect.Struct {
				return nil, errors.New("Unable to access '" + pair[i] + "', '" + pair[i-1] + "' is not a struct")
			}

			field := coreValue.FieldByName(pair[i])
			if field != (reflect.Value{}) {
				value = field.Interface()
				continue
			}

			method := coreValue.MethodByName(pair[i])
			if method == (reflect.Value{}) {
				if corePtrVal.IsValid() {
					method = corePtrVal.MethodByName(pair[i])
				}
				if method == (reflect.Value{}) {
					return nil, errors.New("No method or field '" + pair[i] + "' present on parameter '" + pair[i-1] + "'")
				}
			}

			switch right.(type) {
			case []interface{}:

				givenParams := right.([]interface{})
				params = make([]reflect.Value, len(givenParams))
				for idx, _ := range givenParams {
					params[idx] = reflect.ValueOf(givenParams[idx])
				}

			default:

				if right == nil {
					params = []reflect.Value{}
					break
				}

				params = []reflect.Value{reflect.ValueOf(right.(interface{}))}
			}

			params, err = typeConvertParams(method, params)

			if err != nil {
				return nil, errors.New("Method call failed - '" + pair[0] + "." + pair[1] + "': " + err.Error())
			}

			returned := method.Call(params)
			retLength := len(returned)

			if retLength == 0 {
				return nil, errors.New("Method call '" + pair[i-1] + "." + pair[i] + "' did not return any values.")
			}

			if retLength == 1 {

				value = returned[0].Interface()
				continue
			}

			if retLength == 2 {

				errIface := returned[1].Interface()
				err, validType := errIface.(error)

				if validType && errIface != nil {
					return returned[0].Interface(), err
				}

				value = returned[0].Interface()
				continue
			}

			return nil, errors.New("Method call '" + pair[0] + "." + pair[1] + "' did not return either one value, or a value and an error. Cannot interpret meaning.")
		}

		value = castToFloat32(value)
		return value, nil
	}
}

func separatorStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	var ret []interface{}

	switch left.(type) {
	case []interface{}:
		ret = append(left.([]interface{}), right)
	default:
		ret = []interface{}{left, right}
	}

	return ret, nil
}

func inStage(left interface{}, right interface{}, parameters Parameters) (interface{}, error) {

	for _, value := range right.([]interface{}) {
		if left == value {
			return true, nil
		}
	}
	return false, nil
}

//

func isString(value interface{}) bool {

	switch value.(type) {
	case string:
		return true
	}
	return false
}

func isRegexOrString(value interface{}) bool {

	switch value.(type) {
	case string:
		return true
	case *regexp.Regexp:
		return true
	}
	return false
}

func isBool(value interface{}) bool {
	switch value.(type) {
	case []bool:
		return true
	case bool:
		return true
	}
	return false
}

func isFloat32(value interface{}) bool {

	switch value.(type) {
	case []float32:
		return true
	case float32:
		return true
	}

	return false
}

/*
	Addition usually means between numbers, but can also mean string concat.
	String concat needs one (or both) of the sides to be a string.
*/
func additionTypeCheck(left interface{}, right interface{}) bool {

	if isFloat32(left) && isFloat32(right) {
		return true
	}
	if !isString(left) && !isString(right) {
		return false
	}
	return true
}

/*
	Comparison can either be between numbers, or lexicographic between two strings,
	but never between the two.
*/
func comparatorTypeCheck(left interface{}, right interface{}) bool {

	if isFloat32(left) && isFloat32(right) {
		return true
	}
	if isString(left) && isString(right) {
		return true
	}
	return false
}

func isArray(value interface{}) bool {
	switch value.(type) {
	case []interface{}:
		return true
	}
	return false
}

/*
	Converting a boolean to an interface{} requires an allocation.
	We can use interned bools to avoid this cost.
*/
func boolIface(b bool) interface{} {
	if b {
		return _true
	}
	return _false
}

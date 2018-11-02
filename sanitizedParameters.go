package govaluate

// sanitizedParameters is a wrapper for Parameters that does sanitization as
// parameters are accessed.
type sanitizedParameters struct {
	orig Parameters
}

func (p sanitizedParameters) Get(key string) (interface{}, error) {
	value, err := p.orig.Get(key)
	if err != nil {
		return nil, err
	}

	return castToFloat32(value), nil
}

func castToFloat32(value interface{}) interface{} {
	switch t := value.(type) {
	case uint8:
		return float32(value.(uint8))
	case uint16:
		return float32(value.(uint16))
	case uint32:
		return float32(value.(uint32))
	case uint64:
		return float32(value.(uint64))
	case int8:
		return float32(value.(int8))
	case int16:
		return float32(value.(int16))
	case int32:
		return float32(value.(int32))
	case int64:
		return float32(value.(int64))
	case int:
		return float32(value.(int))
	case float64:
		return float32(value.(float64))

	case []uint8:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []uint16:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []uint32:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []uint64:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []int8:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []int16:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []int32:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []int64:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []int:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	case []float64:
		res := make([]float32, len(t))
		for i, v := range t {
			res[i] = float32(v)
		}
		return res
	}
	return value
}

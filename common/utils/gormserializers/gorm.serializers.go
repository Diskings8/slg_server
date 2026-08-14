package gormserializers

import (
	"context"
	"encoding/json"
	"reflect"

	"gorm.io/gorm/schema"
)

func init() {
	// jsonslice：安全的 JSON 序列化器（nil 切片/映射 → []，合法 JSON）。
	// GORM 默认 JSON 序列化器对 nil 值 + not null 标签会返回空串（serializer.go:106-110），
	// MySQL JSON 列拒收空文档（Error 3140）。所有 JSON 切片字段统一用它。
	schema.RegisterSerializer("jsonslice", JSONSlice{})
}

// JSONSlice 安全的 JSON 序列化器
type JSONSlice struct{}

// Scan 反序列化（同默认 JSON 序列化器）
func (JSONSlice) Scan(ctx context.Context, field *schema.Field, dst reflect.Value, dbValue interface{}) error {
	fieldValue := reflect.New(field.FieldType)
	if dbValue != nil {
		var bytes []byte
		switch v := dbValue.(type) {
		case []byte:
			bytes = v
		case string:
			bytes = []byte(v)
		default:
			b, err := json.Marshal(v)
			if err != nil {
				return err
			}
			bytes = b
		}
		if len(bytes) > 0 {
			if err := json.Unmarshal(bytes, fieldValue.Interface()); err != nil {
				return err
			}
		}
	}
	field.ReflectValueOf(ctx, dst).Set(fieldValue.Elem())
	return nil
}

// Value 序列化：nil 切片/映射 → []；其余 json.Marshal（nil 指针 → "null"，均为合法 JSON）。
func (JSONSlice) Value(ctx context.Context, field *schema.Field, dst reflect.Value, fieldValue interface{}) (interface{}, error) {
	if fieldValue != nil {
		rv := reflect.ValueOf(fieldValue)
		if rv.IsValid() && (rv.Kind() == reflect.Slice || rv.Kind() == reflect.Map) && rv.IsNil() {
			return "[]", nil
		}
	}
	result, err := json.Marshal(fieldValue)
	if err != nil {
		return nil, err
	}
	return string(result), nil
}

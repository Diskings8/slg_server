package util_jsons

import (
	"io"

	jsoniter "github.com/json-iterator/go"
	"server.slg.com/common/utils/util_bytes"
)

// Marshal 编码
func Marshal(in any) ([]byte, error) {
	return jsoniter.Marshal(in)
}

// Unmarshal 解码
func Unmarshal(in []byte, out any) error {
	return jsoniter.Unmarshal(in, out)
}

// UnmarshalFromString 解码字符串
func UnmarshalFromString(in string, out any) error {
	return jsoniter.Unmarshal(util_bytes.ToBytes(in), out)
}

// MarshalIndent 编码并格式化
func MarshalIndent(v any, prefix string, indent string) ([]byte, error) {
	return jsoniter.MarshalIndent(v, prefix, indent)
}

// NewEncoder 创建编码器
func NewEncoder(writer io.Writer) *jsoniter.Encoder {
	return jsoniter.NewEncoder(writer)
}

// NewDecoder 创建解码器
func NewDecoder(reader io.Reader) *jsoniter.Decoder {
	return jsoniter.NewDecoder(reader)
}

// ToJSON 转换json
func ToJSON(data any) string {
	dataBytes, _ := jsoniter.MarshalToString(data)
	return dataBytes
}

// ToJSONBytes 转换[]byte
func ToJSONBytes(data any) []byte {
	dataBytes, _ := jsoniter.Marshal(data)
	return dataBytes
}

package zap_config

import (
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

type SensitiveFieldEncoder struct {
	zapcore.Encoder
	cfg    zapcore.EncoderConfig
	fields []string
}

// EncodeEntry is called for every log line to be emitted so it needs to be
// as efficient as possible so that you don't negate the speed/memory advantages
// of Zap
func (e *SensitiveFieldEncoder) EncodeEntry(
	entry zapcore.Entry,
	fields []zapcore.Field,
) (*buffer.Buffer, error) {
	filtered := make([]zapcore.Field, 0, len(fields))

	for _, field := range fields {
		for _, f := range e.fields {
			if f == field.Key {
				field.String = "[REDACTED]"
				field.Interface = nil
			}
		}
		filtered = append(filtered, field)
	}

	return e.Encoder.EncodeEntry(entry, filtered)
}

func NewSensitiveFieldsEncoder(config zapcore.EncoderConfig, fields []string) zapcore.Encoder {
	encoder := zapcore.NewJSONEncoder(config)
	return &SensitiveFieldEncoder{encoder, config, fields}
}

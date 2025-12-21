package grpcjson

import "encoding/json"


type Codec struct{}

func (Codec) Name() string { return "json" }

func (Codec) Marshal(v any) ([]byte, error) { return json.Marshal(v) }

func (Codec) Unmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }


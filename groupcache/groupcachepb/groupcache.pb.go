package groupcachepb

import (
	"encoding/json"
	"github.com/golang/protobuf/proto"
	"math"
)

var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type GetRequest struct {
	Group             *string `protobuf:"bytes,1,req,name=group" json:"group,omitempty"`
	Key               *string `protobuf:"bytes,2,req,name=key" json:"key,omitempty"`
	XXX_unrecongnized []byte  `json:"-"`
}

func (m *GetRequest) Reset()         { *m = GetRequest{} }
func (m *GetRequest) String() string { return proto.CompactTextString(m) }
func (m *GetRequest) ProtoMessage()  {}

func (m *GetRequest) GetGroup() string {
	if m != nil && m.Group != nil {
		return *m.Group
	}
	return ""
}

func (m *GetRequest) GetKey() string {
	if m != nil && m.Key != nil {
		return *m.Key
	}
	return ""
}

type GetResponse struct {
	Value            []byte   `protobuf:"bytes,1,opt,name=value" json:"value,omitempty"`
	MinuteQps        *float64 `protobuf:"fixed64,2,opt,name=minute_qps" json:"minute_qps,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *GetResponse) Reset()         { *m = GetResponse{} }
func (m *GetResponse) String() string { return proto.CompactTextString(m) }

func (m *GetResponse) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *GetResponse) GetMinuteQps() float64 {
	if m != nil && m.MinuteQps != nil {
		return *m.MinuteQps
	}
	return 0
}
func init() {

}

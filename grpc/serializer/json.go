package serializer

import (
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
)

func ProtobufToJSON(message proto.Message) (string, error) {
	marshaler := jsonpb.Marshaler{
		EnumsAsInts:  true,
		EmitDefaults: true,
		Indent:       " ",
		OrigName:     false,
	}
	return marshaler.MarshalToString(message)
}

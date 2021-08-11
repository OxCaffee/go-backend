package serializer_test

import (
	"../pb"
	"../serializer"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"
	"testing"
)
import "../sample"

func TestFileSerializer(t *testing.T) {
	t.Parallel()
	binaryFile := "../tmp/laptop.bin"
	jsonFile := "../tmp/laptop.json"

	laptop1 := sample.NewLaptop()
	err := serializer.WriteProtobufToBinaryFile(laptop1, binaryFile)
	if err != nil {
		fmt.Errorf("test wrieteprotobuftobin failed: %w", err)
	}

	laptop2 := &pb.Laptop{}
	err = serializer.ReadProtobufToBinaryFile(binaryFile, laptop2)
	if err != nil {
		fmt.Errorf("test ReadProtobufToBinaryFile failed: %w", err)
	}
	if proto.Equal(laptop1, laptop2) == true {
		fmt.Println("equal!")
	}

	err = serializer.WriteProtobufToJSONFile(laptop1, jsonFile)
	require.NoError(t, err)
}

package sapient

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sync"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/dynamicpb"
)

const (
	sapientMessageProtoPath = "sapient_msg/bsi_flex_335_v2_0/sapient_message.proto"
	sapientMessageFullName  = protoreflect.FullName("sapient_msg.bsi_flex_335_v2_0.SapientMessage")
)

//go:embed protos/sapient_msg
var embeddedProtoFiles embed.FS

type ProtoDescriptorSet struct {
	SapientMessage protoreflect.MessageDescriptor
}

var embeddedDescriptors struct {
	once sync.Once
	set  *ProtoDescriptorSet
	err  error
}

func EmbeddedProtoDescriptorSet(ctx context.Context) (*ProtoDescriptorSet, error) {
	embeddedDescriptors.once.Do(func() {
		protoRoot, err := fs.Sub(embeddedProtoFiles, "protos")
		if err != nil {
			embeddedDescriptors.err = err
			return
		}
		embeddedDescriptors.set, embeddedDescriptors.err = CompileProtoDescriptorSet(ctx, protoRoot)
	})
	return embeddedDescriptors.set, embeddedDescriptors.err
}

func CompileProtoDescriptorSet(ctx context.Context, protoRoot fs.FS) (*ProtoDescriptorSet, error) {
	compiler := protocompile.Compiler{
		Resolver: protocompile.WithStandardImports(protoFSResolver{fsys: protoRoot}),
	}
	files, err := compiler.Compile(ctx, sapientMessageProtoPath)
	if err != nil {
		return nil, fmt.Errorf("compile SAPIENT protos: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("compile SAPIENT protos: no files returned")
	}
	desc := files[0].FindDescriptorByName(sapientMessageFullName)
	message, ok := desc.(protoreflect.MessageDescriptor)
	if !ok || message == nil {
		return nil, fmt.Errorf("compile SAPIENT protos: %s message descriptor not found", sapientMessageFullName)
	}
	return &ProtoDescriptorSet{SapientMessage: message}, nil
}

func ParseBinaryMessage(data []byte, descriptors *ProtoDescriptorSet) (Message, error) {
	if descriptors == nil {
		var err error
		descriptors, err = EmbeddedProtoDescriptorSet(context.Background())
		if err != nil {
			return Message{}, err
		}
	}
	if descriptors.SapientMessage == nil {
		return Message{}, fmt.Errorf("SAPIENT protobuf descriptor set is missing SapientMessage")
	}
	dynamic := dynamicpb.NewMessage(descriptors.SapientMessage)
	if err := (proto.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(data, dynamic); err != nil {
		return Message{}, fmt.Errorf("decode SAPIENT binary protobuf: %w", err)
	}
	jsonData, err := (protojson.MarshalOptions{UseProtoNames: false}).Marshal(dynamic)
	if err != nil {
		return Message{}, fmt.Errorf("convert SAPIENT protobuf to JSON preflight form: %w", err)
	}
	message, err := ParseJSONMessage(jsonData)
	if err != nil {
		return Message{}, fmt.Errorf("validate SAPIENT binary protobuf: %w", err)
	}
	return message, nil
}

type protoFSResolver struct {
	fsys fs.FS
}

func (r protoFSResolver) FindFileByPath(path string) (protocompile.SearchResult, error) {
	data, err := fs.ReadFile(r.fsys, path)
	if err != nil {
		return protocompile.SearchResult{}, fmt.Errorf("%w: %s", protoregistry.NotFound, path)
	}
	return protocompile.SearchResult{Source: bytes.NewReader(data)}, nil
}

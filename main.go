package main

import (
	"bytes"
	"io"
	"log"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/vektah/gqlparser/v2/formatter"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/dantedenis/go-proto-gql/pkg/generator"
)

func main() {
	in, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}
	req := &pluginpb.CodeGeneratorRequest{}
	if err := proto.Unmarshal(in, req); err != nil {
		log.Fatal(err)
	}

	files, err := generate(req)
	res := &pluginpb.CodeGeneratorResponse{
		File:              files,
		SupportedFeatures: proto.Uint64(uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)),
	}
	if err != nil {
		res.Error = proto.String(err.Error())
	}

	out, err := proto.Marshal(res)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		log.Fatal(err)
	}
}

func generate(req *pluginpb.CodeGeneratorRequest) (outFiles []*pluginpb.CodeGeneratorResponse_File, err error) {
	var genServiceDesc bool
	var mergePath *string
	var extension = generator.DefaultExtension
	for _, param := range strings.Split(req.GetParameter(), ",") {
		var value string
		if i := strings.Index(param, "="); i >= 0 {
			value, param = param[i+1:], param[0:i]
		}
		switch param {
		case "svc":
			if genServiceDesc, err = strconv.ParseBool(value); err != nil {
				return nil, err
			}
		case "merge":
			v := value
			mergePath = &v
		case "ext":
			extension = strings.Trim(value, ".")
		}
	}
	p, err := protogen.Options{}.New(req)
	if err != nil {
		log.Fatal(err)
	}
	descs, err := generator.CreateDescriptorsFromProto(req)
	if err != nil {
		return nil, err
	}

	merge := mergePath != nil
	gqlDesc, err := generator.NewSchemas(descs, merge, genServiceDesc, p)
	if err != nil {
		return nil, err
	}
	for _, schema := range gqlDesc {
		buff := &bytes.Buffer{}
		formatter.NewFormatter(buff).FormatSchema(schema.AsGraphql())
		protoFileName := schema.FileDescriptors[0].GetName()

		outFiles = append(outFiles, &pluginpb.CodeGeneratorResponse_File{
			Name:    proto.String(resolveGraphqlFilename(protoFileName, mergePath, extension)),
			Content: proto.String(buff.String()),
		})
	}

	return
}

func resolveGraphqlFilename(protoFileName string, merge *string, extension string) string {
	if merge != nil {
		gqlFileName := "schema." + extension
		return path.Join(*merge, gqlFileName)
	}

	return strings.TrimSuffix(protoFileName, path.Ext(protoFileName)) + "." + extension
}

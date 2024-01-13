package proto_store

//type protoFile struct {
//	name    string
//	content []byte
//}

// 	if err := loadProtoFileToRegistry(protoRegistry,
// 		protoFile{
// 			name:    "google/protobuf/timestamp.proto",
// 			content: timestampProtoFile,
// 		},
// 		protoFile{
// 			name:    "marshal.proto",
// 			content: marshalProtoFile,
// 		}); err != nil {
// 		fmt.Println(err)
// 		os.Exit(1)
// 	}

// TODO
//
// - manage to succed using binary proto
// - see the compiler source in that url https://github.com/bufbuild/protocompile/issues/101
// func loadProtoFileToRegistry(pr *protoregistry.Files, protoFiles ...protoFile) error {
// 	for _, p := range protoFiles {
// 		reader := bytes.NewReader(p.content)
// 		handler := reporter.NewHandler(nil)
//
// 		node, err := protoparser.Parse(p.name, reader, handler)
// 		if err != nil {
// 			return fmt.Errorf("parse proto: %w", err)
// 		}
// 		res, err := protoparser.ResultFromAST(node, true /* validate */, handler)
// 		if err != nil {
// 			return fmt.Errorf("convert from AST: %w", err)
// 		}
// 		fd, err := protodesc.NewFile(res.FileDescriptorProto(), pr)
// 		if err != nil {
// 			return fmt.Errorf("convert to FileDescriptor: %w", err)
// 		}
// 		if err := pr.RegisterFile(fd); err != nil {
// 			return fmt.Errorf("register file: %w", err)
// 		}
// 	}
// 	return nil
// }

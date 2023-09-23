# recompile the files regardless of its existence
.PHONY: protos

protos:
	protoc --go_out=. --go_opt=paths=source_relative \
  	--go-grpc_out=. --go-grpc_opt=paths=source_relative \
  	protos/readwrite.proto	

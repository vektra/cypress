log.pb.go: log.proto
	protoc --gogo_out=. -I=.:$(GOPATH)/src/github.com/gogo/protobuf/protobuf:$(GOPATH)/src log.proto
	sed -i ''  's/json:\"-\"/json:\"-\" codec:\"-\"/' log.pb.go


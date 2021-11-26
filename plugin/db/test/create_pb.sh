
cd ../../../..
PROJECT=`pwd`
cd -
TCAPLUS_PB_API_PATH=${PROJECT}/cppsrc/common/extern/tcapluspb/include/tcaplus_pb_api
PROTOC=${PROJECT}/common/extern/protobuf/protoc
PROTOC_GEN_GO=${PROJECT}/gosrc/bingo/codegenerator/protoc-gen-go
PROTO_FILES="db_test.proto tcaplusservice.optionv1.proto"
MODULE_PATH=${PROJECT}/gosrc/
CUR_DIR=`pwd`

rm -f *.pb.go

for proto_file in ${PROTO_FILES}; do	
	${PROTOC} --plugin=${PROTOC_GEN_GO} --go_out=paths=source_relative:.  ${proto_file};
done	

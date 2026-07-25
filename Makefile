.PHONY: proto-game

proto-game:
	@test -s proto/game.proto || (echo "Error: proto/game.proto is missing or empty" && exit 1)
	@mkdir -p shared/genproto
	protoc --go_out=. --go-grpc_out=. proto/game.proto
	@echo "Generated Go code from proto/game.proto"

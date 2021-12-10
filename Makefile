js:
	docker run --rm --name js -p 4222:4222 nats -js

# examples:
# 	go run examples/main.go

server:
	go run cmd/main.go
	
.PHONY: js server
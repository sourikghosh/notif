js:
	docker run --rm -d --name js -p 4222:4222 nats -js

# examples:
# 	go run examples/main.go

trace:
	docker run --rm -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:1.29


server:
	go run cmd/main.go
	
.PHONY: js trace server
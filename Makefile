.PHONY: all run
GOROOT?=/data/server/go-v1.10.2-linux-x64

all: bin/reverse_http_proxy
bin/reverse_http_proxy: main.go handler.go logger.go
	${GOROOT}/bin/go build -o $@ .
run: bin/reverse_http_proxy
	@./bin/reverse_http_proxy

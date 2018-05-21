.PHONY: all run
GOROOT?=/usr/local/go

all: bin/reverse_http_proxy
bin/reverse_http_proxy: main.go handler.go logger.go
	${GOROOT}/bin/go build -o $@ .
run: bin/reverse_http_proxy
	@./bin/reverse_http_proxy

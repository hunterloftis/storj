dist:
	mkdir -p dist
	go build -o dist ./cmd/relay
	go build -o dist ./cmd/send
	go build -o dist ./cmd/receive

.PHONY: dist

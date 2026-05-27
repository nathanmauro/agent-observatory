.PHONY: build dev frontend backend test bench clean install

build: frontend backend

frontend:
	cd web && npm install && npm run build
	cp -r web/dist/* internal/api/static/

backend:
	go build -o agent-observatory ./cmd/agent-observatory/

dev:
	go run ./cmd/agent-observatory/

test:
	go test ./... -count=1

bench:
	go test ./internal/sources/... ./internal/db/ -bench=. -benchmem -count=1

install: build
	install -m 755 agent-observatory $(HOME)/.local/bin/agent-observatory

clean:
	rm -f agent-observatory
	rm -rf web/dist web/node_modules

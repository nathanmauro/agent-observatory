.PHONY: build dev frontend backend clean

build: frontend backend

frontend:
	cd web && npm install && npm run build
	rm -rf internal/api/static/assets internal/api/static/index.html internal/api/static/favicon.svg
	cp -r web/dist/* internal/api/static/

backend:
	go build -o agent-observatory ./cmd/agent-observatory/

dev:
	go run ./cmd/agent-observatory/

clean:
	rm -f agent-observatory
	rm -rf web/dist web/node_modules

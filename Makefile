.PHONY: local railway lint build format

local:
	-colima start
	docker-compose up --build --force-recreate
	@echo "Application started. Backend: http://localhost:8080, Frontend: http://localhost:5173"

lint: format lint-backend lint-frontend

lint-backend:
	cd api-backend && gofmt -l .
	cd api-backend && go vet ./...

lint-frontend:
	cd web-front && pnpm lint

format:
	cd api-backend && gofmt -w .
	cd web-front && pnpm exec prettier --write . --ignore-unknown || true
	@echo "Code formatted"

test-prepare:
	-colima start
	docker-compose up -d test-db
	@echo "Waiting for test-db to be ready..."
	@until docker exec test-db pg_isready -U postgres; do sleep 1; done

railway:
	railway up ./api-backend/ --path-as-root

test: test-prepare
	export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/test_db?sslmode=disable" && cd api-backend && go test ./... -v

.PHONY: local railway lint build format

local:
	-colima start
	docker-compose up --build --force-recreate
	@echo "Application started. Backend: http://localhost:8080, Frontend: http://localhost:5173"

lint: format lint-backend lint-frontend

lint-backend:
	cd railway/api-backend && gofmt -l .
	cd railway/api-backend && go vet ./...

lint-frontend:
	cd web-front && pnpm lint

format:
	cd railway/api-backend && gofmt -w .
	cd web-front && pnpm exec prettier --write . --ignore-unknown || true
	@echo "Code formatted"

railway:
	railway up ./railway/api-backend/ --path-as-root

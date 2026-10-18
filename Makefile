.PHONY: local local-bg clear-local local-fe railway lint lint-backend lint-frontend format test-prepare test test-local stage-frontend build-local-backend build-local install-local install-remote run-local clean-local-build

local:
	-colima start
	docker-compose down
	docker-compose up --build --force-recreate
	@echo "Application started. Backend: http://localhost:3648, Frontend: http://localhost:5173"

local-bg:
	-colima start
	docker-compose down
	docker-compose up --build --force-recreate -d
	@echo "Application started in background. Backend: http://localhost:3648, Frontend: http://localhost:5173"

clear-local:
	docker-compose down -v
	docker volume rm $(docker volume ls -qf dangling=true)
	-colima stop
	@echo "Application stopped and cleared"

local-fe:
	cd front-web && pnpm dev

stage-frontend:
	rm -rf front-web/dist api-backend/frontend/dist
	cd front-web && pnpm build
	@mkdir -p api-backend/frontend/dist
	@cp -R front-web/dist/. api-backend/frontend/dist/

build-local-backend: stage-frontend
	@mkdir -p bin
	cd api-backend && go build -tags frontend -o ../bin/todo-planner-backend-local .

build-local:
	rm -rf bin/todo-planner-backend-local
	$(MAKE) -C front-desktop-widget build-local
	@echo "Local application built at front-desktop-widget/build/TodoPlannerWidget.app"

install-local:
	$(MAKE) -C front-desktop-widget install-local

install-remote:
	$(MAKE) -C front-desktop-widget install-remote

run-local: build-local
	open front-desktop-widget/build/TodoPlannerWidget.app

clean-local-build:
	rm -rf bin/todo-planner-backend-local front-desktop-widget/build

lint: lint-backend lint-frontend

lint-backend:
	@test -z "$$(cd api-backend && gofmt -l .)" || { echo "Go files need formatting. Run make format."; exit 1; }
	cd api-backend && go vet ./...

lint-frontend:
	cd front-web && pnpm lint
	cd front-rn-mobile && pnpm lint

format:
	-cd api-backend && gofmt -w . > /dev/null
	-cd front-web && pnpm exec prettier --write . --ignore-unknown > /dev/null
	-cd front-rn-mobile && pnpm exec prettier --write . --ignore-unknown > /dev/null
	-cd front-desktop-widget && swift format --in-place --recursive Sources/ Tests/ > /dev/null
	@echo "Code formatted"

test-prepare:
	-colima start
	docker-compose up -d test-db
	@echo "Waiting for test-db to be ready..."
	@until docker exec test-db pg_isready -U postgres; do sleep 1; done

railway:
	railway up ./api-backend/ --path-as-root

test: test-prepare
	export TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/test_db?sslmode=disable" && cd api-backend && go test ./...

test-local:
	EMBEDDED_TEST_DATABASE=true cd api-backend && go test -count=1 ./...

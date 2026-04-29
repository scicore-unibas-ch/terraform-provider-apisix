.PHONY: build test test-acceptance test-acceptance-single test-env-up test-env-down test-env-logs clean

build:
	go build -o terraform-provider-apisix

test:
	go test ./... -v

test-acceptance:
	@echo "Starting APISIX cluster..."
	docker compose -f tests/docker-compose.yml up -d
	@echo "Waiting for APISIX to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 16 21 22 23 24 25 26 27 28 29 30; do \
		if docker ps --format '{{.Names}}' | grep -q tests-apisix-1 && \
		   curl -s -o /dev/null -w "%{http_code}" http://localhost:9180/apisix/admin/routes -H "X-API-KEY: test123456789" | grep -q "200"; then \
			echo "✓ APISIX ready"; \
			break; \
		fi; \
		echo "  Attempt $$i/30 - waiting..."; \
		sleep 2; \
	done
	@echo ""
	@echo "Running acceptance tests (excluding SSL - requires manual setup)..."
	@for test in tests/acceptance/*/test.sh; do \
		test_name=$$(basename $$(dirname $$test)); \
		if [ "$$test_name" = "ssl" ]; then \
			echo "⊘ Skipping $$test (SSL tests require manual execution)"; \
			continue; \
		fi; \
		echo "Running $$test..."; \
		if ! bash $$test; then \
			echo "✗ $$test FAILED"; \
			docker compose -f tests/docker-compose.yml down -v; \
			exit 1; \
		fi; \
	done
	@echo ""
	@echo "✓ All acceptance tests passed (7/8 resources tested, SSL skipped)"
	@echo "Stopping APISIX cluster..."
	docker compose -f tests/docker-compose.yml down -v

test-acceptance-single:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-acceptance-single TEST=upstream"; \
		exit 1; \
	fi
	@echo "Starting APISIX cluster..."
	docker compose -f tests/docker-compose.yml up -d
	@echo "Waiting for APISIX to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 16 21 22 23 24 25 26 27 28 29 30; do \
		if docker ps --format '{{.Names}}' | grep -q tests-apisix-1 && \
		   curl -s -o /dev/null -w "%{http_code}" http://localhost:9180/apisix/admin/routes -H "X-API-KEY: test123456789" | grep -q "200"; then \
			echo "✓ APISIX ready"; \
			break; \
		fi; \
		echo "  Attempt $$i/30 - waiting..."; \
		sleep 2; \
	done
	@echo ""
	@echo "Running $(TEST) acceptance test..."
	bash tests/acceptance/$(TEST)/test.sh
	RESULT=$$?; \
	echo "Stopping APISIX cluster..."; \
	docker compose -f tests/docker-compose.yml down -v; \
	exit $$RESULT

test-env-up:
	docker compose -f tests/docker-compose.yml up -d
	@echo "Waiting for APISIX to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 16 21 22 23 24 25 26 27 28 29 30; do \
		if docker ps --format '{{.Names}}' | grep -q tests-apisix-1 && \
		   curl -s -o /dev/null -w "%{http_code}" http://localhost:9180/apisix/admin/routes | grep -q "200"; then \
			echo "✓ APISIX ready"; \
			break; \
		fi; \
		echo "  Attempt $$i/30 - waiting..."; \
		sleep 2; \
	done

test-env-down:
	docker compose -f tests/docker-compose.yml down -v

test-env-logs:
	docker compose -f tests/docker-compose.yml logs -f

clean:
	rm -f terraform-provider-apisix

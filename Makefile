.PHONY: build test clean test-env-up test-env-down test-env-logs test-acceptance test-acceptance-single

build:
	go build -o terraform-provider-apisix

test:
	go test ./... -v

test-acceptance:
	@echo "Running acceptance tests..."
	@for test in tests/acceptance/*/test.sh; do \
		echo "Running $$test..."; \
		if ! bash $$test; then \
			echo "✗ $$test FAILED"; \
			exit 1; \
		fi; \
	done
	@echo "✓ All acceptance tests passed"

test-acceptance-single:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-acceptance-single TEST=upstream"; \
		exit 1; \
	fi
	bash tests/acceptance/$(TEST)/test.sh

test-env-up:
	docker compose -f tests/docker-compose.yml up -d
	@echo "Waiting for APISIX to be ready..."
	@for i in 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30; do \
		if docker ps --format '{{.Names}}' | grep -q apisix && \
		   curl -s -o /dev/null -w "%{http_code}" http://localhost:9180/apisix/admin/routes | grep -q "200\|401\|403"; then \
			echo "✓ APISIX Admin API is ready"; \
			break; \
		fi; \
		echo "  Attempt $$i/30 - waiting..."; \
		sleep 2; \
	done
	@echo "APISIX is ready!"

test-env-down:
	docker compose -f tests/docker-compose.yml down

test-env-logs:
	docker compose -f tests/docker-compose.yml logs -f

clean:
	rm -f terraform-provider-apisix

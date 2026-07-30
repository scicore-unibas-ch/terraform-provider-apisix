.PHONY: build test test-acceptance test-acceptance-single test-env-up test-env-down test-env-logs wait-for-apisix clean

# Admin API of the test stack (tests/docker-compose.yml). The key must match
# the one baked into that compose file.
APISIX_BASE_URL ?= http://localhost:9180/apisix/admin
APISIX_ADMIN_KEY ?= test123456789
APISIX_WAIT_ATTEMPTS ?= 30

build:
	go build -o terraform-provider-apisix

test:
	go test ./... -v

test-acceptance:
	@echo "Starting APISIX cluster..."
	docker compose -f tests/docker-compose.yml up -d
	@$(MAKE) --no-print-directory wait-for-apisix
	@echo ""
	@echo "Running acceptance tests..."
	@for test in tests/acceptance/*/test.sh; do \
		echo "Running $$test..."; \
		if ! bash $$test; then \
			echo "✗ $$test FAILED"; \
			docker compose -f tests/docker-compose.yml down -v; \
			exit 1; \
		fi; \
	done
	@echo ""
	@echo "✓ All acceptance tests passed"
	@echo "Stopping APISIX cluster..."
	docker compose -f tests/docker-compose.yml down -v

test-acceptance-single:
	@if [ -z "$(TEST)" ]; then \
		echo "Usage: make test-acceptance-single TEST=upstream"; \
		exit 1; \
	fi
	@echo "Starting APISIX cluster..."
	docker compose -f tests/docker-compose.yml up -d
	@$(MAKE) --no-print-directory wait-for-apisix
	@echo ""
	@echo "Running $(TEST) acceptance test..."
	bash tests/acceptance/$(TEST)/test.sh
	RESULT=$$?; \
	echo "Stopping APISIX cluster..."; \
	docker compose -f tests/docker-compose.yml down -v; \
	exit $$RESULT

test-env-up:
	docker compose -f tests/docker-compose.yml up -d
	@$(MAKE) --no-print-directory wait-for-apisix

# Block until the admin API answers. Every target that needs a live stack
# depends on this one rather than carrying its own copy of the loop: the copies
# drifted before, and the one in test-env-up spent 60s polling a 401 because it
# omitted the API key.
wait-for-apisix:
	@echo "Waiting for APISIX to be ready..."
	@for i in $$(seq 1 $(APISIX_WAIT_ATTEMPTS)); do \
		if docker ps --format '{{.Names}}' | grep -q tests-apisix-1 && \
		   [ "$$(curl -s -o /dev/null -w '%{http_code}' -H 'X-API-KEY: $(APISIX_ADMIN_KEY)' $(APISIX_BASE_URL)/routes)" = "200" ]; then \
			echo "✓ APISIX ready"; \
			exit 0; \
		fi; \
		echo "  Attempt $$i/$(APISIX_WAIT_ATTEMPTS) - waiting..."; \
		sleep 2; \
	done; \
	echo "✗ APISIX did not become ready after $(APISIX_WAIT_ATTEMPTS) attempts"; \
	docker compose -f tests/docker-compose.yml logs --tail 30 apisix; \
	exit 1

test-env-down:
	docker compose -f tests/docker-compose.yml down -v

test-env-logs:
	docker compose -f tests/docker-compose.yml logs -f

clean:
	rm -f terraform-provider-apisix

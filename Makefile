# Default target
.DEFAULT_GOAL := help

DOCKER_COMPOSE=docker compose -f deployments/docker-compose.yml -p simple-route
GOLANG=go
ECHO=echo

# Help
.PHONY: help compose-run compose-config compose-show compose-stop run
help:
	@echo "Makefile commands:"
	@echo "  make compose-run         - Docker compose run the application stack"
	@echo "  make compose-config      - Docker compose configure"
	@echo "  make compose-show        - Docker compose shown process"
	@echo "  make compose-stop        - Docker compose stop and remove stack"
	@echo "  make compose-attach      - Attach to service"
	@echo "  make run                 - Run golang app (should be run inside this container environment or any available go in your path)"

compose-run:
	$(DOCKER_COMPOSE) up --build -d

# Configure
compose-config:
	$(DOCKER_COMPOSE) config

# Shown process
compose-show:
	$(DOCKER_COMPOSE) ps -a

# Attach
compose-attach:
	$(DOCKER_COMPOSE) exec $(service) $(tty)

# Clean up
compose-stop:
	$(DOCKER_COMPOSE) stop && $(DOCKER_COMPOSE) rm -f

run:
	$(GOLANG) run ./example
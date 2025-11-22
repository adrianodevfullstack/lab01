.PHONY: test test-coverage test-watch build run clean help

# Executa os testes
test:
	docker-compose run --rm test

# Executa os testes com geração de relatório de cobertura
test-coverage:
	docker-compose run --rm test-coverage

# Executa os testes localmente (sem Docker)
test-local:
	go test -v -cover ./...

# Executa os testes localmente com cobertura
test-coverage-local:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Cobertura gerada em coverage.html"

# Build da aplicação
build:
	docker-compose build app

# Executa a aplicação
run:
	docker-compose up app

# Limpa containers e volumes
clean:
	docker-compose down -v
	docker system prune -f

# Mostra ajuda
help:
	@echo "Comandos disponíveis:"
	@echo "  make test              - Executa os testes no Docker"
	@echo "  make test-coverage     - Executa testes com relatório de cobertura"
	@echo "  make test-local        - Executa testes localmente"
	@echo "  make test-coverage-local - Executa testes localmente com cobertura"
	@echo "  make build             - Build da aplicação"
	@echo "  make run               - Executa a aplicação"
	@echo "  make clean             - Limpa containers e volumes"
	@echo "  make help              - Mostra esta ajuda"


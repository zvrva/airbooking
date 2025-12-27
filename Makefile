generate-api:
\t@./scripts/generate.sh

up:
\tdocker-compose up -d

down:
\tdocker-compose down

cov:
\tgo test ./...

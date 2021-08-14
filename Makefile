format:
	sh ./scripts/format.sh

vet:
	go vet ./...
	shadow ./...
	sh ./scripts/errcheck.sh
	sh ./scripts/staticcheck.sh

genkey:
	go run ./cmd/keygen/main.go

passgen:
	go run ./cmd/passgen/main.go -pass ${pass}

uuidgen:
	go run ./cmd/uuidgen/main.go

keygen:
	go run ./cmd/keygen/main.go -dir ${dir}
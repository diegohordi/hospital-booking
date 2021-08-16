format:
	sh ./scripts/format.sh
	go mod tidy

vet:
	go vet ./...
	sh ./scripts/shadow.sh
	sh ./scripts/errcheck.sh
	sh ./scripts/staticcheck.sh
	go mod tidy

genkey:
	go run ./cmd/keygen/main.go

passgen:
	go run ./cmd/passgen/main.go -pass ${pass}

uuidgen:
	go run ./cmd/uuidgen/main.go

keygen:
	go run ./cmd/keygen/main.go -dir ${dir}

run_test:
	docker-compose up --build --abort-on-container-exit hospital_booking_backend_test

run:
	docker-compose --profile deploy up --build -d

stop:
	docker-compose down -v
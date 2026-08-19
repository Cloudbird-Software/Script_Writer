.PHONY: setup fmt lint arch test build check

setup:  ; go mod download
fmt:    ; gofmt -w .
lint:   ; test -z "$$(gofmt -l .)" && go vet ./...
# GO-3 边界检查：main 包只允许出现在 cmd/；internal 包不得反向 import cmd/
arch:
	@offenders=$$(go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./... | grep -v '^github.com/Cloudbird-Software/Script_Writer/cmd/' | sed '/^$$/d'); \
	if [ -n "$$offenders" ]; then \
		echo "::error::main 包越界（只允许 cmd/）：$$offenders"; exit 1; \
	fi; \
	if go list -deps ./internal/... 2>/dev/null | grep -q 'Script_Writer/cmd'; then \
		echo "::error::禁止 internal 包 import cmd/"; exit 1; \
	fi
test:   ; go test -race ./...
build:  ; go build -o bin/songguard ./cmd/songguard
check:  lint arch test

tfpgen:
	go build -o tfpgen

clean:
	rm -rf generated

fmtcheck:
	gofmt -s -l .

fmt:
	gofmt -s .

test:
	go test -v ./...

.PHONY: clean fmt fmtcheck test

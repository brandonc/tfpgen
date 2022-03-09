tfpgen:
	go build -o tfpgen

.PHONY: clean
clean:
	rm -rf generated

.PHONY: benchmark

benchmark:
	go test -v -benchmem -run=^$$ -bench "[{Do}|{Get}]$$" -benchtime=1s

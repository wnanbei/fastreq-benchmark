.PHONY: benchmark

benchmark:
	go test -v -benchmem -run=^$$ -bench "[{Do}|{Get}]$$" -cpu=4,8 -benchtime=2s

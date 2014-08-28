
debug:
	go build  -gcflags "-N -l"
	go install

clean:
	rm -f gavelin *~ *.o

testbuild:
	go test -c -gcflags "-N -l" -v

test:
	go test -v

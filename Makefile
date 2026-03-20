.PHONY: build test proto run bench clean

export CGO_ENABLED=1

build:
	mkdir -p engine/build bin
	clang++ -shared -fPIC -g -O2 -std=c++17 -install_name @rpath/libvektor_engine.dylib -I engine/include engine/src/markov_chain.cpp engine/src/decision_engine.cpp engine/src/engine.cpp -o engine/build/libvektor_engine.dylib
	go build -o bin/proxy cmd/proxy/main.go
	go build -o bin/coordinator cmd/coordinator/main.go
	go build -o bin/bench cmd/bench/main.go

test:
	mkdir -p bin
	clang++ -g -O2 -std=c++17 -fsanitize=thread -I engine/include engine/tests/ring_buffer_test.cpp -o bin/ring_buffer_test && ./bin/ring_buffer_test
	clang++ -g -O2 -std=c++17 -fsanitize=thread -I engine/include engine/src/markov_chain.cpp engine/tests/markov_chain_test.cpp -o bin/markov_chain_test && ./bin/markov_chain_test
	clang++ -g -O2 -std=c++17 -fsanitize=thread -I engine/include engine/src/markov_chain.cpp engine/src/decision_engine.cpp engine/src/engine.cpp engine/tests/decision_engine_test.cpp -o bin/decision_engine_test && ./bin/decision_engine_test
	go test -v ./...

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/vektor.proto

run:
	docker-compose up --build

bench:
	./bin/bench -mode baseline -trace trace.bin -duration 5s
	./bin/bench -mode vektor -trace trace.bin -duration 5s

clean:
	rm -rf engine/build/*
	rm -rf bin/*
	find . -name "*.dylib" -type f -delete
	find . -name "*.so" -type f -delete

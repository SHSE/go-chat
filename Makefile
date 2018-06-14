clean:
	rm -rf build
	mkdir build

build: clean
	vgo build -o build/chat ./cmd/chat

bench: build
	/bin/sh bench.sh

bench-docker:
	/bin/sh bench-docker.sh

run: build
	CHAT_PORT=3000 ./build/chat

test:
	vgo test -v -cover -timeout=3s ./cmd/chat

archive: clean
	git archive --format=tar.gz --prefix=chat/ -o build/chat.tar.gz HEAD

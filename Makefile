BIN=./bin
CLIENT=s83
SERVER=s83d
COVER=cover.out
COVERHTML=cover.html

# docker
SERVERTAG=s83/server

all: test build

test:
	go vet ./... 
	go test -race ./...

cover:
	go test -coverprofile ${BIN}/${COVER} ./...
	go tool cover -html=${BIN}/${COVER} -o ${BIN}/${COVERHTML}

build: build-client build-server

build-client:
	go build -o ${BIN}/${CLIENT} ./cmd/client/...

build-server:
	go build -o ${BIN}/${SERVER} ./cmd/server/...

serve: build-server
	cd ${BIN}; mkdir -p store; ./${SERVER}

clean:
	rm -f ${BIN}/${CLIENT}
	rm -f ${BIN}/${SERVER}
	rm -f ${BIN}/${COVER}
	rm -f ${BIN}/${COVERHTML}

docker-build: test docker-build-server

docker-build-server:
	docker build -t ${SERVERTAG} -f Dockerfile.server .

docker-serve: docker-build-server
	docker run --rm -it -p 8080:8080 ${SERVERTAG}

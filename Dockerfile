FROM golang:1.25.1 AS build
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o tetris-server ./cmd/server
FROM scratch
COPY --from=build /build/tetris-server /usr/bin/tetris-server
EXPOSE 9000
ENTRYPOINT ["/usr/bin/tetris-server"]

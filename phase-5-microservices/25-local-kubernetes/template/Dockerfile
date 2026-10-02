# syntax=docker/dockerfile:1

# Build stage: compiles a static binary. CGO is disabled so the result has no dynamic library
# dependencies, which is what lets the final stage be distroless (no shell, no package manager,
# no libc beyond what's statically linked in).
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/service ./cmd/service

# Final stage: distroless, non-root by default. No shell means no `docker exec sh` for
# debugging — a deliberate production tradeoff (smaller attack surface) you may want to swap
# for a `-debug` distroless variant while developing this practice.
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/service /service
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/service"]

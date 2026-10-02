FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY main.go ./
RUN go build -o /paymentprovider .

FROM alpine:3.21
COPY --from=build /paymentprovider /paymentprovider
EXPOSE 9090
ENTRYPOINT ["/paymentprovider"]

FROM golang:1.18 as builder
RUN mkdir /build
ADD . /build/
WORKDIR /build
RUN CGO_ENABLED=0 go build -a -o app .

FROM alpine:3.11.3
COPY --from=builder /build/app .

ENTRYPOINT [ "./app" ]

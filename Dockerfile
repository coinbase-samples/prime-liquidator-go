FROM public.ecr.aws/docker/library/golang:latest as builder

ARG CACHEBUST=1

RUN mkdir -p /build
WORKDIR /build
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main cmd/server/main.go

FROM scratch

COPY --from=builder /build/main /main
COPY --from=builder /etc/ssl/certs /etc/ssl/certs

CMD ["/main"]

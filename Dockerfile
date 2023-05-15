FROM golang:1.19 AS builder

# Get the binary compilation products of Block service and Meta service.
WORKDIR /app
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go build -o /app/shamrock-block ./cmd/block
RUN go build -o /app/shamrock-meta ./cmd/meta

# Docker image for packaging block storage services.
FROM alpine as block
COPY --from=builder /app/shamrock-block /app/shamrock-block
ENV PATH=/app:$PATH
WORKDIR /app
CMD ["shamrock-block"]

# Docker image for packaging meta information services.
FROM alpine as meta
COPY --from=builder /app/shamrock-meta /app/shamrock-meta
ENV PATH=/app/:$PATH
WORKDIR /app
CMD ["shamrock-meta"]
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /waller .


FROM alpine:latest
WORKDIR /app
RUN apk update && apk add --no-cache libheif-tools exiv2 imagemagick
COPY --from=builder /waller .
COPY static ./static
CMD ["./waller"]

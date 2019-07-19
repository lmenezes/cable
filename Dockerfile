FROM golang:1.12 as build

WORKDIR /go/src/cable
COPY . .

ENV GO111MODULE on
RUN go build -o /cable -v go/cmd/cable.go

# Now copy it into our base image.
FROM gcr.io/distroless/base
COPY --from=build  /cable /
CMD ["/cable"]
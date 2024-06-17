FROM --platform=$BUILDPLATFORM golang:1.22 as build
ARG TARGETARCH

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -o /go/bin/app

FROM gcr.io/distroless/static-debian11

COPY --from=build /go/bin/app /
COPY --from=build /go/src/app/front /front
USER nonroot
CMD ["/app"]
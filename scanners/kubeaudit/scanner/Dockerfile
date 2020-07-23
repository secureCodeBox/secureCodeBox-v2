FROM golang:1.13-alpine
WORKDIR /go/src/github.com/Shopify/kubeaudit
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kubeaudit -v cmd/main.go

FROM alpine
COPY --from=0 /go/src/github.com/Shopify/kubeaudit/kubeaudit .
COPY wrapper.sh /wrapper.sh
ENTRYPOINT ["./kubeaudit"]
CMD ["all"]

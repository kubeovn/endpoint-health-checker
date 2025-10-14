VERSION ?= $(shell cat VERSION | tr -d '\n')

.PHONY: build
build:
	go mod tidy
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$(VERSION)" -o dist/images/endpoint_health_checker ./main.go

.PHONY: build-image
build-image:
	docker build -t endpoint_health_checker:$(VERSION) -f dist/images/Dockerfile .

.PHONY: install-chart
install-chart:
	helm install endpoint-health-checker ./chart --namespace kube-system

.PHONY: uninstall-chart
uninstall-chart:
	helm uninstall endpoint-health-checker --namespace kube-system

# kind load docker-image --name kube-ovn endpoint_health_checker:v0.1.0
# docker tag endpoint_health_checker:v0.1.0 build-harbor.alauda.cn/3rdparty/endpoint_health_checker:v0.1.0
# kind load docker-image --name kube-ovn build-harbor.alauda.cn/3rdparty/endpoint_health_checker:v0.1.0
# kubectl delete pod -l app=endpoint-health-checker -n kube-system
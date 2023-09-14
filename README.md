# registry-proxy

本项目帮助您在 Kubernetes 集群中愉快的拉取国外的镜像，从让人无法忍受的网络限制或龟速网速中解脱出来。

首先需要感谢 [Docker Proxy](https://dockerproxy.com/) 提供的镜像代理服务，本项目才能得以实现。

## 实现原理

使用 MutatingValidationWebhook 修改 Pod 的容器镜像，如果容器镜像属于国外镜像，例如：gcr.io/xxx/xxx、k8s.registry.io/xxx/xxx，那么将容器镜像修改成 dockerproxy 代理镜像。

## 快速安装

⚠️ 前提：需要安装 cert-manager，如果没有安装，可以参考 [Install cert-manager](https://cert-manager.io/docs/installation/) 安装。

这里提供快速安装的方式：

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# 代理地址
kubectl apply -f https://ghproxy.com/https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

**安装 registry-proxy**

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml

# 代理地址
kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml
```

## 示例

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml

# 代理地址
kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml
```

## 卸载

```bash
kubectl delete -f https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml

# 代理地址
kubectl delete -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml
```

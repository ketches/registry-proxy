# registry-proxy

Registry Proxy，帮助您在 Kubernetes 集群中愉快地拉取国外容器镜像（仅限公有镜像）。

目前支持的境外镜像仓库：

- docker.io
- ghcr.io
- gcr.io
- k8s.gcr.io
- registry.k8s.io
- quay.io
- mcr.microsoft.com

PS：感谢 [Docker Proxy](https://dockerproxy.com/) 提供的镜像代理服务，本项目才能得以实现。💕

## 实现原理

使用 Mutating Webhook 准入控制器实现。 当集群中 Pod 创建时，Mutating Webhook 的工作流程如下：

1、判断 Pod 是否属于排除的命名空间，如果是，结束流程；
2、判断 Pod 是否属于包含的命名空间，如果不是，结束流程；
3、依次判断 Pod 中的容器镜像是否属于包含的镜像仓库，如果是，替换为 Docker Proxy 代理镜像；

![202309201040207](https://pding.oss-cn-hangzhou.aliyuncs.com/images/202309201040207.png)

## 代理参考

### ****Docker Hub 官方镜像代理****

- 常规镜像代理
    - stilleshan/frpc:latest => dockerproxy.com/stilleshan/frpc:latest

- 根镜像代理
    - nginx:latest => dockerproxy.com/library/nginx:latest

### GitHub Container Registry

- 常规镜像代理
    - ghcr.io/username/image:tag => ghcr.dockerproxy.com/username/image:tag

### Google Container Registry

- 常规镜像代理
    - gcr.io/username/image:tag => gcr.dockerproxy.com/username/image:tag

### Google Kubernetes

- 常规镜像代理
    - k8s.gcr.io/username/image:tag => k8s.dockerproxy.com/username/image:tag
    - registry.k8s.io/username/image:tag => k8s.dockerproxy.com/username/image:tag

- 根镜像代理
    - k8s.gcr.io/coredns:1.6.5 => k8s.dockerproxy.com/coredns:1.6.5
    - registry.k8s.io/coredns:1.6.5 => k8s.dockerproxy.com/coredns:1.6.5

### Quay.io

- 常规镜像代理
    - quay.io/username/image:tag => quay.dockerproxy.com/username/image:tag

### Microsoft Artifact Registry

- 常规镜像代理
    - mcr.microsoft.com/azure-cognitive-services/diagnostic:latest => mcr.dockerproxy.com/azure-cognitive-services/diagnostic:latest

## 快速安装

**安装 cert-manager**

*如果集群中已经安装了 cert-manager，可以跳过这一步。*

这里提供快速安装的方式：

```bash
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml

# 代理地址
kubectl apply -f https://ghproxy.com/https://github.com/cert-manager/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

> 官方文档： [Install cert-manager](https://cert-manager.io/docs/installation/)。

**安装 registry-proxy**

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml

# 代理地址
kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml
```

**配置**

三个配置参数，以下给出默认配置，只有在命名空间范围内的 Pod，且 Pod 镜像在 Registry 范围内，容器镜像才会修改为 Docker Proxy 代理镜像。

1. excludeNamespaces：["kube-system", "kube-public", "kube-node-lease"]
2. includeNamespaces: ["*"]
3. includeRegistries: ["docker.io", "ghcr.io", "gcr.io", "k8s.gcr.io", "registry.k8s.io", "quay.io", "mcr.microsoft.com"]

通过 ConfigMap 修改默认配置，修改会实时生效。

示例：限定代理命名空间 default、dev 和 staging 中 docker.io 的镜像。

```bash
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: registry-proxy-config
  namespace: registry-proxy
data:
  config.yaml: |
    excludeNamespaces:
    - kube-system
    - kube-public
    - kube-node-lease
    includeNamespaces:
    - dev
    - staging
    includeRegistries:
    - "docker.io"
EOF
```

## 使用方式

## 示例

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml

# 代理地址
kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml
```

## 卸载&清理

**卸载 registry-proxy**

```bash
kubectl delete -f https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml

# 代理地址
kubectl delete -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml
```

**清理示例**

```bash
kubectl delete -f https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml

# 代理地址
kubectl delete -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml
```

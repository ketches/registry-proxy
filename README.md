# registry-proxy

在 Kubernetes 集群中部署 Registry Proxy，自动帮助您使用镜像代理服务拉取新创建的 Pod 中的外网容器镜像（仅限公有镜像）。

**适用场景**

1. 无法拉取例如 K8s (registry.k8s.io) 、谷歌 (gcr.io) 等镜像；
2. 龟速拉取例如 GitHub(ghcr.io)、RedHat(quay.io) 等镜像；

**代理清单**

默认镜像代理服务支持的外网镜像仓库：

- docker.io
- registry.k8s.io
- quay.io
- ghcr.io
- gcr.io
- k8s.gcr.io
- docker.cloudsmith.io

## 快速安装

1. **安装 cert-manager**

   如果集群中已经安装了 *cert-manager*，可以跳过这一步。这里提供快速安装的方式：

   ```bash
   kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml
   
   # 代理地址
   kubectl apply -f https://ghproxy.com/https://github.com/cert-manager/cert-manager/releases/download/v1.13.2/cert-manager.yaml
   ```

   > 官方文档： [Install cert-manager](https://cert-manager.io/docs/installation/)。

2. **安装 registry-proxy**

   ```bash
   kubectl apply -f https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml
   
   # 代理地址
   kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/deploy/manifests.yaml
   ```

## 配置

registry-proxy 安装后自动创建 ConfigMap `registry-proxy-config`，ConfigMap 内容为默认配置，可以通过修改 ConfigMap 来修改默认配置。

默认配置：

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: registry-proxy-config
  namespace: registry-proxy
data:
  config.yaml: |
    proxies:
      docker.io: docker.ketches.cn
      registry.k8s.io: k8s.ketches.cn
      quay.io: quay.ketches.cn
      ghcr.io: ghcr.ketches.cn
      gcr.io: gcr.ketches.cn
      k8s.gcr.io: k8s-gcr.ketches.cn
      docker.cloudsmith.io: cloudsmith.ketches.cn
    excludeNamespaces:
    - kube-system
    - kube-public
    - kube-node-lease
    includeNamespaces:
    - *
```

> Notes：
> 1. 默认使用 [ketches/cloudflare-registry-proxy](https://github.com/ketches/cloudflare-registry-proxy) 镜像代理服务；
> 2. 默认排除 `kube-system`、`kube-public`、`kube-node-lease` 命名空间下的 Pod 容器镜像代理；
> 3. 修改上述配置实时生效，无需重启 registry-proxy；
> 4. 可以自定义代理地址，例如：`docker.io: docker.m.daocloud.io`;
> 5. 可以去除代理地址，免去代理；
> 6. 可以增加代理地址，例如：`mcr.microsoft.com: mcr.dockerproxy.com`；
> 7. 可以通过向 [ketches/cloudflare-registry-proxy](https://github.com/ketches/cloudflare-registry-proxy) 项目 [提交 Issue](https://github.com/ketches/cloudflare-registry-proxy/issues/new) 来申请添加新的国外镜像代理服务

## 实现原理

使用 Mutating Webhook 准入控制器实现。 当集群中 Pod 创建时，Mutating Webhook 的工作流程如下：

1. 判断 Pod 是否属于排除的命名空间，如果是，结束流程；
2. 判断 Pod 是否属于包含的命名空间，如果不是，结束流程；
3. 依次判断 Pod 中的容器镜像是否匹配代理仓库，如果是，替换为代理镜像；

![202311071243391](https://pding.oss-cn-hangzhou.aliyuncs.com/images/202311071243391.png)

## 使用示例

使用 Docker 镜像 nginx 创建一个 Pod：

```bash
kubectl apply -f https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml

# 代理地址
kubectl apply -f https://ghproxy.com/https://raw.githubusercontent.com/ketches/registry-proxy/master/examples/dockerhub-nginx.yaml
```

示例中的 Pod 镜像为 `nginx:latest`，经过 registry-proxy 自动代理后，容器镜像变为 `docker.ketches.cn/library/nginx:latest`。

验证：

```bash
kubectl get pod dockerhub-nginx -o=jsonpath='{.spec.containers[*].image}'
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

## 代理参考

### Docker Hub 镜像代理

常规镜像代理
- ketches/registry-proxy:latest => docker.ketches.cn/ketches/registry-proxy:latest

根镜像代理
- nginx:latest => docker.ketches.cn/library/nginx:latest

### Kubernetes 镜像代理

常规镜像代理
- registry.k8s.io/ingress-nginx/controller:v1.8.2 => k8s.ketches.cn/ingress-nginx/controller:v1.8.2

根镜像代理
- registry.k8s.io/pause:3.9 => k8s.ketches.cn/pause:3.9

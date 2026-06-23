# 服务网关

## 需求

平台需要一个统一的南北向流量入口，集中承担：

- 把外部请求路由到控制面（Platform）与数据面（工作区 / 在线服务）的目标 Service；
- 在入口层完成数据面 JWT 鉴权；
- 提供限流、超时、熔断、重试等横切流量治理；
- **声明式、CRD 化**配置，可被各组件 controller 在工作负载 namespace 内动态派生路由；
- 原生支持 gRPC / HTTP2（推理服务常用）。

## 技术选型

选用 **[Envoy Gateway](https://gateway.envoyproxy.io/)**（基于 Kubernetes [Gateway API](https://gateway-api.sigs.k8s.io/)）。它如何满足上述需求：Gateway API 是官方 Ingress 继任者，配置完全 CRD 化、支持跨 namespace 派生路由；Envoy 数据面原生 gRPC / HTTP2；`SecurityPolicy` / `BackendTrafficPolicy` 扩展分别覆盖鉴权与流量治理；零外部依赖。

## 资源模型

```
GatewayClass (envoy-gateway)
  └── Gateway (axisml-gateway, in axisml-infra ns)  Listener: HTTP(80)/HTTPS(443)
        │  allowedRoutes.namespaces: 放行接入工作负载所在 namespace
        └── HTTPRoute（静态 / 派生）→ 目标 ClusterIP Service
```

- **Gateway**：单一 `axisml-gateway` 承载全部路由，由 `axisml-infra` chart 提供。
- **静态 HTTPRoute**：由调用方 chart 一同发布，对接控制面服务对外接口。
- **派生 HTTPRoute**：调用方 controller 在工作负载 namespace 内创建 `HTTPRoute` / `SecurityPolicy` / `BackendTrafficPolicy`，`parentRefs` 指向 `axisml-gateway`；`ReferenceGrant` 仅在跨 namespace `backendRef` 授权场景使用。

## 对外契约

| 能力 | 资源 | 说明 |
| --- | --- | --- |
| 认证鉴权 | `SecurityPolicy`（附加到 Gateway / HTTPRoute） | JWT 验证（issuer + JWKS）· OIDC 集成 · ExtAuth · per-Service（`targetRefs`）。具体 IdP 由调用方决定，本功能只保证能力就位 |
| 流量控制 | `BackendTrafficPolicy` | 限流 · 熔断 · 超时 / 重试 · 负载均衡 |

本功能只提供 `Gateway` 与 listener 能力，不感知业务语义、不内置用户态鉴权策略。部署形态见 [deployment.md §5](../../../docs/system_design/deployment.md#5-控制面-deployment)。

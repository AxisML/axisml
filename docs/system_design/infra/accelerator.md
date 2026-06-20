# 加速器管理

## 需求

把节点上的物理 GPU 暴露为 Kubernetes 可调度资源并保持可观测，要求：把 GPU 暴露为 extended resource `nvidia.com/gpu` 供调度器分配；自动化驱动 / 设备插件 / 运行时集成的生命周期（免节点手工装驱动）；按 GPU 型号给节点打标以支持亲和；导出 GPU 利用率 / 显存 / 温度等指标。

## 技术选型

选用 **[NVIDIA GPU Operator](https://github.com/NVIDIA/gpu-operator)**。理由：Kubernetes 原生 GPU 管理的事实标准，驱动 / 工具链容器化满足免手工装驱动；DCGM Exporter 与监控栈天然集成，开箱导出 GPU 指标。

## 组件构成

| 组件 | 职责 |
| --- | --- |
| GPU Driver Container | 容器化 NVIDIA 驱动，自动安装与升级 |
| NVIDIA Container Toolkit | 容器运行时集成，使容器可访问 GPU |
| Device Plugin | 向调度器报告 `nvidia.com/gpu` 资源 |
| DCGM Exporter | 导出 GPU 利用率 / 显存 / 温度 / 功耗等 Prometheus 指标 |
| GPU Feature Discovery | 自动为节点打标（GPU 型号、驱动版本等） |
| MIG Manager | A100 / H100 多实例分区管理（按需启用） |

## 调度契约

- 业务 Pod 申请 GPU 使用资源名 `nvidia.com/gpu`；实际调度由 [调度与配额](scheduler.md) 完成，本功能不做调度决策。
- 节点标签 `nvidia.com/gpu.product`（如 `A100-SXM4-80GB`）可做 nodeSelector / affinity。
- DCGM Exporter 的 `/metrics` 由 [监控](monitoring.md) 自动采集。

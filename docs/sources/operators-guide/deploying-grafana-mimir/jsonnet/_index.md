---
title: "Deploying Grafana Mimir with Jsonnet and Tanka"
menuTitle: "Deploying with Jsonnet and Tanka"
description: "Learn how to deploy Grafana Mimir on Kubernetes with Jsonnet and Tanka."
weight: 10
keywords:
  - Mimir deployment
  - Kubernetes
  - Jsonnet
  - Tanka
---

# Deploying Grafana Mimir with Jsonnet and Tanka

Grafana Labs publishes [Jsonnet](https://jsonnet.org/) files that you can use to deploy Grafana Mimir in [microservices mode]({{< relref "../../architecture/deployment-modes/index.md#microservices-mode" >}}).
Jsonnet files are located in the [Mimir repository](https://github.com/grafana/mimir/tree/main/operations/mimir).

{{< section >}}

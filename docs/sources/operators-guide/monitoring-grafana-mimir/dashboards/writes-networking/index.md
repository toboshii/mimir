---
title: "Grafana Mimir Writes networking dashboard"
menuTitle: "Writes networking"
description: "View an example Writes networking dashboard."
weight: 190
aliases:
  - ../../visualizing-metrics/dashboards/writes-networking/
---

# Grafana Mimir Writes networking dashboard

The Writes networking dashboard shows receive/transmit bandwidth, inflight requests, and TCP connections.
The dashboard isolates each service on the write path into its own section and displays the order in which a write request flows.

This dashboard requires [additional resources metrics]({{< relref "../../requirements.md#additional-resources-metrics" >}}).

## Example

The following example shows a Writes networking dashboard from a demo cluster.

![Grafana Mimir writes networking dashboard](mimir-writes-networking.png)

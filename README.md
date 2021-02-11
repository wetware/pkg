# Wetware

[![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/wetware/ww)
[![Go Report Card](https://goreportcard.com/badge/github.com/SentimensRG/ctx?style=flat-square)](https://goreportcard.com/report/github.com/wetware/ww)
![tests](https://github.com/wetware/ww/workflows/Go/badge.svg)

**Wetware is the language of the cloud.**  It is an alternative to [Kubernetes](https://kubernetes.io/), [Mesos](https://mesos.apache.org/) and [OpenShift](https://www.openshift.com/) that turns any group of networked computers -- including cloud-based instances -- into a programmable IaaS/PaaS cluster.

**Developers** use wetware to write distributed applications that can be instantly ported from a single laptop to the datacenter, cloud, or a hybrid of both.

**Managers** love Wetware for its organizational benefits, which include a unified API for coordinating access to resources across teams, cloud-agnostic services that avoid vendor lock-in, and a small learning curve that onboards developers faster.

But there's more.  Wetware is a full-fledged distributed systems language, complete with a batteries-included standard library, and a rich ecosystem that your team will never outgrow.

- [Wetware](#wetware)
  - [Quickstart](#quickstart)
  - [Motivation](#motivation)
    - [Fear, uncertainty, and declarative config](#fear-uncertainty-and-declarative-config)
    - [The Wetware Way](#the-wetware-way)
  - [How it Works](#how-it-works)
  - [Getting Started](#getting-started)
  - [Documentation and Support](#documentation-and-support)
  - [Provisional License](#provisional-license)
    - [Copyright Notice](#copyright-notice)
    - [Note Regarding Provisional License](#note-regarding-provisional-license)

## Quickstart

TODO

<!-- See our official [Getting Started](https://wetware.dev/quickstart) guide if this is your first time working with Wetware.

For all other documentation, including installation, worked examples, and support, refer to the [documentation section](#documentation-and-support).

[Try it](https://wetware.dev/try) in your browser. -->

## Motivation

### Fear, uncertainty, and declarative config

Existing IaaS/PaaS like Kubernetes try to hide cluster state behind *declarative config*, often written in a high-level markup language like YAML, TOML or JSON.  Instead of *programming* your infrastructure, you *declare* the desired state of your cluster in a configuration file, and the software tries to figure out a way to reach that state.  Declarative approaches work well for applications like database queries, but cause serious problems in a IaaS/PaaS setting.

The problem is that declarative syntax gets datacenters exactly backwards.  In a datacenter or cloud environment, you need to keep track of two things at all times:

1. What is the current configuration of my cluster? (state)
2. How do I get to the desired state?  (strategy)

But with IaaS/PaaS systems like Kubernetes and Mesos, both 1 & 2 are burried under multiple layers of configuration, indirection and abstract interfaces.  And although you know where you want to go (*i.e.*, the state described by your YAML config), you can't be sure where are right now, nor how your config translates into execution.

When you encounter a problem, it's hard to diagnose what went wrong.  All you can really do is grapple with configuration, restart services, and make educated guesses.  Worse, existing IaaS/PaaS systems are incomprehensibly complex, and frequently degenerate into inconsistent states that force you to reboot the entire cluster, especially as you scale.  When that happens you lose valuable debugging information that could have prevented the next incident.

### The Wetware Way

Wetware breaks this vicious cycle by turning IaaS/PaaS on its head.  Instead of static configuration, you're given a powerful language for querying and programming your cluster.  Wetware comes with a REPL to interactively run code on your cluster, high-performance datastructures and synchronization primitives that make concurrency simple, and a rich standard library for writing distributed systems, batteries included.

Drawing inspiration from proven paradigms such as UNIX, Wetware will feel familiar to junior devs and CTOs alike, empowering them to _finally_ treat infrastructure as code.

For managers and devops, Wetware's intuitive cluster API provides a clear, accountable and safe interface between the various engineering roles in a technology company, increasing iteration speed and reducing time-to-value.

## How it Works

<!-- TODO: technical overview (three-layer model) -->

## Getting Started

TODO

## Documentation and Support

TODO

<!-- TODO:  point people to docs, discourse, slack channel and paid support options -->

<!--
Possible names for paid-support agencies:

- Cephalogic
- Cortech  ("Cortech support"  has a nice ring to it)
- ...

-->

## Provisional License

Until this Provisional License and accompanying Copyright Notice are removed,
the terms listed under the [Copyright Notice](https://github.com/wetware/ww#copyright-notice) section shall govern use of all
intellectual proprty contained in any repository under https://github.com/wetware,
and shall take precedence over any other licenses contained therein, to the maximum extent allowed by controlling law.

### Copyright Notice

Copyright 2020 Louis Thibault
All rights reserved.

Authors:
- 2020 Louis Thibault

### Note Regarding Provisional License

For the avoidance of doubt:

Wetware's ambition is to foster an **open, collaborative community**, for the
betterment of our profession.  As such, the authors pledge to replace the provisional license
with retroactive, open-source license as soon as a suitable candidate has been identified.

In the meantime, we welcome feedback and discussion over licensing, copyright and community.

Please refer issues https://github.com/wetware/ww/issues/1 & https://github.com/wetware/ww/issues/2 to track progress on this front.

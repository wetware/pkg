# Wetware

[![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/wetware/ww)
[![Go Report Card](https://goreportcard.com/badge/github.com/SentimensRG/ctx?style=flat-square)](https://goreportcard.com/report/github.com/wetware/ww)
![tests](https://github.com/wetware/ww/workflows/Go/badge.svg)

Wetware is a programming language for the cloud.  It's like adding an interactive shell to [Mesos](https://mesos.apache.org/), or a systems language for [Kubernetes](https://kubernetes.io/).

But that's where the comparison ends.  Wetware is a whole new way of writing cloud applications that is simpler, safer, more reliable and more productive than anything you've seen before.

- [Wetware](#wetware)
  - [Quickstart](#quickstart)
  - [Motivation](#motivation)
    - [The Problem](#the-problem)
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

>I am a developer and I find Kubernetes frustrating. To me, its documentation is confusing and scattered among too many places (best example: overlay networks). I have read multiple books and gazillions of articles and yet I have the feeling that **I am lacking the bigger picture.**
>
>    — [cygned](https://news.ycombinator.com/item?id=18955326)

Writing distributed software is hard, and existing tools aren't helping.  Existing cloud management systems are slow, cumbersome, and confusing by design.

To make matters worse, they're [notoriously fragile](https://k8s.af/), require specialized teams to operate, and have a steep learning curve.

### The Problem

Instead of empowering you to react to your environment, existing cloud management systems hide problems behind various layers of routing, caching and indirection.  By hiding too much, they take power *away* from developers, and corner them into writing broken systems.  They don't just hide the boring bits, they hide distribution itself.

### The Wetware Way

Wetware takes the opposite approach.  It embraces distribution, bringing problems to the surface, and equipping you with a rich standard library to handle failures gracefully.

Drawing inspiration from proven paradigms such as Lisp and UNIX, Wetware empowers you to write software that is understandable, scalable and fault-tolerant from the ground up, without compromizing on usability and ergonomics.

Wetware's design abides by the following principles:

- **Small is better than large**

  Wetware is distributed as a single static binary that weighs less than a mobile app.  It uses network connections sparingly, employs low-chatter protocols, and is optimized for small CPU and memory usage, making it ideal for datacenters as well as IoT.

  As a language, Wetware features simple syntax, concise idioms, and a small set of built-in features to ensures your codebase stays lean, clear, and performant.
  
- **Simple is better than complex**
  
  TODO

  <!-- // handful of moving parts => understandable/adoptable by all -->

  <!-- With airtight abstractions and only a handful of moving parts, Wetware is understandable and adoptable by all. -->


<!-- - **Libraries, not frameworks**   -->
  
- **Dynamic programming over static configuration**

  >It's just YAML until you need multi-tenancy, auto-scaling, security auditing, automated os/container patching, multi-tenant self-service route/ingress management, multi-tenant self-service logging, monitoring and alerting, multi-tenant self-service databases, and so on...
  >
  > — [ukoki](https://news.ycombinator.com/item?id=18963198)

  Cloud management systems often configure behaviors using using languages for _data_.  In YAML, TOML or JSON-based configuration, behaviors are implicit, which obscures dataflow, hides dependencies and introduces vulnerabilities.  Developers struggle to reuse configuration, discover useful settings, extend working systems, and track down config errors.  Templates only compound the problem by spreading configuration state across multiple locations, introducing new dependencies, and increasing cognitive overhead.
  
  Wetware solves this issue at the root by representing behavior as code.  With its code-as-data philosophy, powerful macro system, and fast-staring REPL, Wetware unlocks your most powerful tools:  composition, abstraction and testing.
  
- **Systems, not hacks**

  TODO

  <!-- // harmony, symbiosis, gestalt, data-layer as unification



  Wetware combines multiple technologies that complement each other.

  Wetware's peer-to-peer cluster protocol is self-healing and [antifragile](https://en.wikipedia.org/wiki/Antifragility), its BitSwap protocol efficiently streams terabytes of data across your cluster, securely, and its location-aware DHT ensures you're always fetching data from the nearest source, avoiding egress costs in hybrid and multicloud setups.

  All of this happens out-of-the box, with zero additional configuration or user intervention, making Wetware truly greater than the sum of its parts. -->

- **Ergonomics matter**.

  // TODO  <!-- iso environments, REPL, herokuness, zeroconf -->

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

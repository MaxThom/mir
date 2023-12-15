
<p align="center">
<img src="./images/mir_alpha.png" alt="mir logo" style="width:428px;"/>
</p>

<h1 align="center" style="font-size: 48px;">Mir IoT Hub</h1>
<h3 align="center">Mir hub is the ultimate IoT hub solution for tommorow's interconnected world
</h3>
<h3 align="center">Develop easier. Connect faster. Scale quicker.
</h3>

<br/>
<p align="center">
  <a href="https://github.com/maxthom/mir/blob/master/LICENSE">
    <img src="https://img.shields.io/github/license/maxthom/mir">
  </a>
  <img src="https://img.shields.io/github/repo-size/maxthom/mir">
</p>

# What is Mir Iot Hub?

Enable highly secure and reliable communication between your Internet of Things (IoT) application and the devices it manages. Mur IoT Hub provides a cloud-hosted solution back end to connect virtually any device. Extend your solution from the cloud to the edge with per-device authentication, built-in device management, device observability, device data, and scaled provisioning.

Mir IoT Hub, act as your command center:

- processes telemetry and commands with two ways communication
- automatically generate dashboards to observe data
- uses device twin for configuration management
- over the air update
- lightweight and infinitely scalable

# Content

- [Features](#features)
- [Documentation](#documentation)
- [Installation](#installation)
- [Getting started](#getting-started)
  - [Device side apps](#device-side-apps)
  - [Server side apps](#server-side-apps)
- [Modules](#modules)
  - [User defined module](#user-defined-module)
  - [Configuration module](#configuration-module)
  - [Observability module](#observability-module)
  - [User templated-data module](#user-templated-data-module)
- [Road map](#roadmap)
- [License](#license)

# Features

# Documentation

# Installation

# Getting started

## Device side apps

## Server side apps

# Modules

## User defined module

## Configuration module

## Observability module

## User templated-data module

# Roadmap

- [ ] Server side sdk to interact with the hub
  - [ ] api that receive the bytes and must be deserialize using protoc code gen
  - [ ] offers utils such as api routes, disk or cli ways to upload a bpb
  - [ ] go-sdk
  - [ ] rust-sdk
  - [ ] python-sdk
- [ ] ProtoProxy, the templated data engine
  - [ ] proto to grafana dashboard
  - [ ] store to timeseries db
- [ ] TwinHub, the configuration module
- [ ] Client side sdk to interact with Mir
  - [ ] go-sdk
  - [ ] rust-sdk
  - [ ] python-sdk
- [ ] MirUI, the web ui
- [ ] The obersavility module
- [ ] Swarm, the device simulator sdk
- [ ] MirOperator, Kubernetes operator that manage the deployments and scaling
- [ ] Installation methods
  - [ ] docker & compose
  - [ ] helm chart
  - [ ] helm chart with MirOperator

# License

Source code for MirHub is licensed under a MIT license

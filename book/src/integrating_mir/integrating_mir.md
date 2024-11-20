# Integrating Mir

## DeviceSDK

With the DeviceSDK you will be able to integrate your devices or softwares
from different origin into one platform to operate your fleet as one.
Mir provides different communication methods such as telemetry, commands and configuration.
Moreover, the DeviceSDK offers common requirements such as local storage if network is out,
data encryption for security, and more.

Mir will let you focus on your device code by providing an all battery included experience:
 - data visualization tools
 - alerting system
 - device observability
 - device commands & control
 - and more...

 ## ModuleSDK

 The ModuleSDK is the other side of the coin. It provides a way to extend the system server side with custom modules.
 This allows you to integrate Mir with your own ecosystem, databases, etc.
 For example, you could integrate devices with your ERP system or else.
 You can also create modules to expand the capabilities of the system such as creating a weekly PDF reports module.

 The ModuleSDK provices a set of functions to liscen to device data, send commands and interact with the system
 such as creating or updating devices. Moreover, the Mir system offers events which are
 server side computed information. For example, a device output heartbeats and from that,
 the core module compute if a device comes online or goes offline. When this happens, an event is generated
 so you can act on an newly online or offline device.

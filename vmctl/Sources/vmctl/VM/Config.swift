import Foundation
import Virtualization

struct DiskConfig: Codable {
  var path: String
  var readOnly: Bool
}

struct Config: Codable {
  var cpuCount: Int
  var memoryMB: UInt64
  var noGraphics: Bool?
  var displayWidth: Int
  var displayHeight: Int
  var additionalDisks: [DiskConfig]?
  var macAddress: String?
  var tty: Bool?

  static func load(from url: URL) throws -> Config {
    var config = try JSONDecoder().decode(Config.self, from: try Data(contentsOf: url))
    config.additionalDisks = config.additionalDisks?.map {
      var disk = $0
      disk.path = URL(string: disk.path, relativeTo: url)!.resolvingSymlinksInPath().path
      return disk
    }
    return config
  }

  private func configureDisplay(_ cfg: VZVirtualMachineConfiguration) {
    let graphics = VZMacGraphicsDeviceConfiguration()
    graphics.displays = [
      VZMacGraphicsDisplayConfiguration(
        for: NSScreen.main!,
        sizeInPoints: NSSize(width: self.displayWidth, height: self.displayHeight)
      )
    ]
    cfg.graphicsDevices = [graphics]
  }

  private func configureStorage(_ cfg: VZVirtualMachineConfiguration, bundle: VMBundle) throws {
    let attachment = try VZDiskImageStorageDeviceAttachment(
      url: bundle.diskImageURL, readOnly: false)
    let disk = VZVirtioBlockDeviceConfiguration(attachment: attachment)
    cfg.storageDevices = [disk]

    for diskConfig in self.additionalDisks ?? [] {
      let attachment = try VZDiskImageStorageDeviceAttachment(
        url: URL(fileURLWithPath: diskConfig.path),
        readOnly: diskConfig.readOnly)
      let disk = VZVirtioBlockDeviceConfiguration(attachment: attachment)
      cfg.storageDevices.append(disk)
    }
  }

  func instantiate(bundle: VMBundle) throws
    -> VZVirtualMachineConfiguration
  {
    let cfg = VZVirtualMachineConfiguration()

    let platform = VZMacPlatformConfiguration()
    platform.auxiliaryStorage = VZMacAuxiliaryStorage(contentsOf: bundle.auxURL)
    platform.hardwareModel = VZMacHardwareModel(
      dataRepresentation: try Data(contentsOf: bundle.modelURL))!
    platform.machineIdentifier = VZMacMachineIdentifier(
      dataRepresentation: try Data(contentsOf: bundle.identifierURL))!

    cfg.platform = platform
    cfg.cpuCount = self.cpuCount
    cfg.memorySize = self.memoryMB * 1024 * 1024
    cfg.bootLoader = VZMacOSBootLoader()

    self.configureDisplay(cfg)
    try self.configureStorage(cfg, bundle: bundle)

    let networkDevice = VZVirtioNetworkDeviceConfiguration()
    networkDevice.attachment = VZNATNetworkDeviceAttachment()
    if let macAddress = self.macAddress {
      guard let macAddress = VZMACAddress(string: macAddress) else {
        fatalError("invalid mac address: \(macAddress)")
      }
      networkDevice.macAddress = macAddress
    }
    cfg.networkDevices = [networkDevice]

    cfg.pointingDevices = [VZUSBScreenCoordinatePointingDeviceConfiguration()]
    cfg.keyboards = [VZUSBKeyboardConfiguration()]
    cfg.entropyDevices = [VZVirtioEntropyDeviceConfiguration()]
    cfg.memoryBalloonDevices = [VZVirtioTraditionalMemoryBalloonDeviceConfiguration()]

    if tty ?? true {
      let dev = VZVirtioConsoleDeviceSerialPortConfiguration()
      dev.attachment = VZFileHandleSerialPortAttachment(
        fileHandleForReading: FileHandle.standardInput,
        fileHandleForWriting: FileHandle.standardOutput
      )
      cfg.serialPorts = [dev]
    }

    try cfg.validate()
    return cfg
  }
}

import Foundation
import Virtualization

struct VMBundle {
  let url: URL

  init(path: String) throws {
    url = URL(fileURLWithPath: path, isDirectory: true)
    try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
  }

  var diskImageURL: URL { url.appendingPathComponent("disk.img") }

  var auxURL: URL { url.appendingPathComponent("aux.img") }

  var modelURL: URL { url.appendingPathComponent("model.dat") }

  var identifierURL: URL { url.appendingPathComponent("identifier.dat") }

  func setup(from model: VZMacHardwareModel, diskSizeMB: UInt64) throws {
    try "".write(to: self.diskImageURL, atomically: true, encoding: .utf8)
    let diskImage = try FileHandle(forWritingTo: self.diskImageURL)
    try diskImage.truncate(atOffset: 0)
    try diskImage.truncate(atOffset: diskSizeMB * 1024 * 1024)

    _ = try VZMacAuxiliaryStorage(
      creatingStorageAt: self.auxURL, hardwareModel: model, options: .allowOverwrite)
    try model.dataRepresentation.write(to: self.modelURL)

    let id = VZMacMachineIdentifier()
    try id.dataRepresentation.write(to: self.identifierURL)
  }
}

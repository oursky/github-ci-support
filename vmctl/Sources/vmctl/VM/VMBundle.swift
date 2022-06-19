import Foundation
import Virtualization

struct VMBundle {
  let url: URL

  init(url: URL) throws {
    self.url = url
    try FileManager.default.createDirectory(at: url, withIntermediateDirectories: true)
  }

  var diskImageURL: URL { url.appendingPathComponent("disk.img").resolvingSymlinksInPath() }

  var auxURL: URL { url.appendingPathComponent("aux.img").resolvingSymlinksInPath() }

  var modelURL: URL { url.appendingPathComponent("model.dat").resolvingSymlinksInPath() }

  var identifierURL: URL { url.appendingPathComponent("identifier.dat").resolvingSymlinksInPath() }

  func clone(to url: URL) throws -> VMBundle {
    let manager = FileManager.default
    try? manager.removeItem(at: url)

    let target = try VMBundle(url: url)
    try manager.copyItem(at: self.diskImageURL, to: target.diskImageURL)
    try manager.copyItem(at: self.auxURL, to: target.auxURL)
    try manager.copyItem(at: self.modelURL, to: target.modelURL)
    try manager.copyItem(at: self.identifierURL, to: target.identifierURL)

    return target
  }

  func setup(from model: VZMacHardwareModel, diskSizeMB: UInt64) throws {
    try Data().write(to: self.diskImageURL)
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

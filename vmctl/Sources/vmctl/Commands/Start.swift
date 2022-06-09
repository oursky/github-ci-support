import ArgumentParser
import Foundation
import Virtualization

struct Start: AsyncParsableCommand {
  @Option(help: "Path to VM config")
  var config: String

  @Option(help: "Path to VM bundle")
  var bundle: String

  @Flag(help: "Boot into recovery")
  var recovery = false

  @MainActor
  func run() async throws {
    let bundle = try VMBundle(url: URL(fileURLWithPath: self.bundle, isDirectory: true))
    let config = try Config.load(from: URL(fileURLWithPath: self.config))

    print("starting VM...")
    let instance = Instance(config: try config.instantiate(bundle: bundle))
    try await instance.start(recovery: recovery)

    if config.noGraphics ?? true {
      dispatchMain()
    } else {
      runVMApp(instance: instance)
    }
  }
}

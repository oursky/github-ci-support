import ArgumentParser
import Foundation
import Virtualization

struct Start: AsyncParsableCommand {
  @Option(help: "Path to VM config")
  var config: String

  @Option(help: "Path to VM bundle")
  var bundle: String

  @MainActor
  func run() async throws {
    let bundle = try VMBundle(path: self.bundle)
    let config = try Config.load(from: URL(fileURLWithPath: self.config))

    print("starting VM...")
    let instance = Instance(config: try config.instantiate(bundle: bundle))
    try await instance.start()

    if config.noGraphics ?? true {
      dispatchMain()
    } else {
      runVMApp(instance: instance)
    }
  }
}

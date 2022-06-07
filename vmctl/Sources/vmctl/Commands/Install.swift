import ArgumentParser
import Foundation
import Virtualization

struct Install: AsyncParsableCommand {
  @Option(help: "Path to VM config")
  var config: String

  @Option(help: "Path to VM bundle")
  var bundle: String

  @Option(help: "Path to IPSW image.")
  var ipsw: String

  @MainActor
  func run() async throws {
    let bundle = try VMBundle(path: self.bundle)
    let ipsw = try await withCheckedThrowingContinuation { complete in
      VZMacOSRestoreImage.load(from: URL(fileURLWithPath: self.ipsw)) { complete.resume(with: $0) }
    }
    let config = try Config.load(from: URL(fileURLWithPath: self.config))

    guard let requirements = ipsw.mostFeaturefulSupportedConfiguration else {
      fatalError("cannot extract hardward config from IPSW")
    }
    try bundle.setup(from: requirements.hardwareModel, diskSizeMB: 50 * 1024)

    let installer = VMInstaller(from: ipsw, config: try config.instantiate(bundle: bundle))

    let observer = installer.progress.observe(\.fractionCompleted, options: [.new]) {
      (_, change) in
      print("progress: \((change.newValue! * 100).rounded())%.")
    }
    print("installing...")
    try await installer.install()
    observer.invalidate()
  }
}

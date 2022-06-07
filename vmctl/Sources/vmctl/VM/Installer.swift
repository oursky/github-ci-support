import Foundation
import Virtualization

@MainActor
struct VMInstaller {
  let vm: VZVirtualMachine
  private let installer: VZMacOSInstaller

  var progress: Progress { self.installer.progress }

  init(from image: VZMacOSRestoreImage, config: VZVirtualMachineConfiguration) {
    self.vm = VZVirtualMachine(configuration: config, queue: DispatchQueue.main)
    self.installer = VZMacOSInstaller(virtualMachine: vm, restoringFromImageAt: image.url)
  }

  func install() async throws {
    try await withCheckedThrowingContinuation { complete in
      self.installer.install { complete.resume(with: $0) }
    }
  }
}

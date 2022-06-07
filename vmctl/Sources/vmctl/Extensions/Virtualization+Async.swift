import Foundation
import Virtualization

extension VZMacOSRestoreImage {
  static func load(from url: URL) async throws -> VZMacOSRestoreImage {
    return try await withCheckedThrowingContinuation { complete in
      self.load(from: url) { complete.resume(with: $0) }
    }
  }
}

extension VZMacOSInstaller {
  @MainActor
  func install() async throws {
    return try await withCheckedThrowingContinuation { complete in
      self.install { complete.resume(with: $0) }
    }
  }
}

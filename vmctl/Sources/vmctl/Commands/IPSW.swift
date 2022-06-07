import ArgumentParser
import Foundation
import Virtualization

struct IPSW: AsyncParsableCommand {
  @MainActor
  func run() async throws {
    let ipsw = try await withCheckedThrowingContinuation { complete in
      VZMacOSRestoreImage.fetchLatestSupported { complete.resume(with: $0) }
    }
    print(ipsw.url)
  }
}

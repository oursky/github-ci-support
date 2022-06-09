import ArgumentParser
import Foundation
import Virtualization

struct Clone: AsyncParsableCommand {
  @Argument(help: "Source VM path")
  var from: String

  @Argument(help: "New VM path")
  var to: String

  @MainActor
  func run() async throws {
    let bundle = try VMBundle(url: URL(fileURLWithPath: self.from, isDirectory: true))
    let cloneURL = URL(fileURLWithPath: to)
    print("cloning VM to \(cloneURL.path)...")
    _ = try bundle.clone(to: cloneURL)
  }
}

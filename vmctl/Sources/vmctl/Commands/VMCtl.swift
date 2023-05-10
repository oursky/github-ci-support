import ArgumentParser
import Foundation

@main
struct VMCtl: AsyncParsableCommand {
  static var configuration = CommandConfiguration(
    subcommands: [Install.self, Start.self, IPSW.self, Clone.self]
  )

  static func main() async {
    do {
      var command = try parseAsRoot()
      if var asyncCommand = command as? AsyncParsableCommand {
        try await asyncCommand.run()
      } else {
        try command.run()
      }
    } catch {
      exit(withError: error)
    }
  }
}

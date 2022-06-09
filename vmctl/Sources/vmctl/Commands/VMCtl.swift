import ArgumentParser

@main
struct VMCtl: AsyncParsableCommand {
  static var configuration = CommandConfiguration(
    subcommands: [Install.self, Start.self, IPSW.self, Clone.self]
  )
}

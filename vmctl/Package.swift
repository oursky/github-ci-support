// swift-tools-version: 5.6
// The swift-tools-version declares the minimum version of Swift required to build this package.

import PackageDescription

let package = Package(
  name: "vmctl",
  platforms: [.macOS("12.3")],
  products: [
    .executable(name: "vmctl", targets: ["vmctl"])
  ],
  dependencies: [
    .package(url: "https://github.com/apple/swift-argument-parser.git", from: "1.1.2"),
    .package(url: "https://github.com/soffes/HotKey", from: "0.1.2"),
  ],
  targets: [
    .executableTarget(
      name: "vmctl",
      dependencies: [
        .product(name: "ArgumentParser", package: "swift-argument-parser"),
        .product(name: "HotKey", package: "HotKey"),
      ])
  ]
)

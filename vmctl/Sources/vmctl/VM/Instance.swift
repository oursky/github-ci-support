import Combine
import Foundation
import Virtualization

@objc private protocol _VZVirtualMachine {
  @objc(_startWithOptions:completionHandler:)
  func _start(with options: _VZVirtualMachineStartOptions) async throws
}

@objc private protocol _VZVirtualMachineStartOptions {
  init()
  var panicAction: Bool { get set }
  var stopInIBootStage1: Bool { get set }
  var stopInIBootStage2: Bool { get set }
  var bootMacOSRecovery: Bool { get set }
  var forceDFU: Bool { get set }
}

class Instance: NSObject, VZVirtualMachineDelegate {
  let vm: VZVirtualMachine
  var keyScript: [KeyScriptInstr]?

  let onStop = PassthroughSubject<(), Never>()

  @MainActor
  init(config: VZVirtualMachineConfiguration) {
    self.vm = VZVirtualMachine(configuration: config, queue: DispatchQueue.main)
    super.init()
    self.vm.delegate = self
  }

  @MainActor
  func start(recovery: Bool = false) async throws {
    Task.detached { @MainActor in
      if #available(macOS 13.0, *) {
        let options: VZMacOSVirtualMachineStartOptions = VZMacOSVirtualMachineStartOptions()
        options.startUpFromMacOSRecovery = recovery
        try await self.vm.start(options: options)
      } else {
        // https://github.com/saagarjha/VirtualApple/blob/8231082e026211d992568fdececc6f47609669ac/VirtualApple/VirtualMachine.swift#L135
        let vm = unsafeBitCast(self.vm, to: _VZVirtualMachine.self)
        let options = unsafeBitCast(
          NSClassFromString("_VZVirtualMachineStartOptions")!,
          to: _VZVirtualMachineStartOptions.Type.self
        ).init()
        options.bootMacOSRecovery = recovery
        try await vm._start(with: options)
      }
    }
  }

  func loadKeyScript(fromURL url: URL) throws {
    let data = try Data(contentsOf: url)
    self.keyScript = try JSONDecoder().decode([KeyScriptInstr].self, from: data)
  }

  func guestDidStop(_ virtualMachine: VZVirtualMachine) {
    print("VM stopped.")
    self.onStop.send()
  }

  func virtualMachine(_ virtualMachine: VZVirtualMachine, didStopWithError error: Error) {
    print("VM failed: \(error)")
    self.onStop.send()
  }

  func virtualMachine(
    _ virtualMachine: VZVirtualMachine, networkDevice: VZNetworkDevice,
    attachmentWasDisconnectedWithError error: Error
  ) {
    print("VM network disconnected: \(error)")
    try? self.vm.requestStop()
  }
}

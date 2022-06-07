import Combine
import Foundation
import Virtualization

class Instance: NSObject, VZVirtualMachineDelegate {
  let vm: VZVirtualMachine

  let onStop = PassthroughSubject<(), Never>()

  @MainActor
  init(config: VZVirtualMachineConfiguration) {
    self.vm = VZVirtualMachine(configuration: config, queue: DispatchQueue.main)
    super.init()
    self.vm.delegate = self
  }

  @MainActor
  func start() async throws {
    Task.detached {
      try await withCheckedThrowingContinuation { complete in
        DispatchQueue.main.async {
          self.vm.start { complete.resume(with: $0) }
        }
      }
    }
  }

  func guestDidStop(_ virtualMachine: VZVirtualMachine) {
    print("VM stopped.")
    self.onStop.send()
  }

  func virtualMachine(_ virtualMachine: VZVirtualMachine, didStopWithError error: Error) {
    print("VM failed: \(error)")
    self.onStop.send()
  }
}

import SwiftUI
import Virtualization

struct VZVirtualMachineViewRepresentable: NSViewRepresentable {
  typealias NSViewType = VZVirtualMachineView

  var virtualMachine: VZVirtualMachine?
  var capturesSystemKeys: Bool?

  func makeNSView(context: Context) -> VZVirtualMachineView {
    let view = VZVirtualMachineView()
    view.virtualMachine = self.virtualMachine
    view.capturesSystemKeys = self.capturesSystemKeys ?? false
    return view
  }

  func updateNSView(_ view: VZVirtualMachineView, context: Context) {
    view.virtualMachine = self.virtualMachine
    view.capturesSystemKeys = self.capturesSystemKeys ?? false
  }
}

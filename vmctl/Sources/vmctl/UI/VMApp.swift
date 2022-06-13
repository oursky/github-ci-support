import Combine
import SwiftUI
import Virtualization

private class AppDelegate: NSObject, NSApplicationDelegate {
  func applicationDidFinishLaunching(_ notification: Notification) {
    NSApplication.shared.setActivationPolicy(.regular)
    NSApplication.shared.activate(ignoringOtherApps: true)
    NSWindow.allowsAutomaticWindowTabbing = false
  }
  func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return true
  }
}

private var _instance: Instance!
private struct VMApp: App {
  @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
  @StateObject private var scriptRunner = KeyScriptRunner(script: _instance.keyScript)

  var body: some Scene {
    WindowGroup {
      VStack {
        HStack {
          Text(self.scriptRunner.instrText)
        }.frame(minHeight: 20, maxHeight: 20)
        VZVirtualMachineViewRepresentable(
          view: self.$scriptRunner.view,
          virtualMachine: _instance.vm,
          capturesSystemKeys: true
        )
      }
      .frame(minWidth: 800, minHeight: 600, alignment: .center)
      .onReceive(_instance.onStop) { _ in
        NSApplication.shared.terminate(nil)
      }
    }
  }
}

private struct VZVirtualMachineViewRepresentable: NSViewRepresentable {
  typealias NSViewType = VZVirtualMachineView

  @Binding var view: VZVirtualMachineView?

  var virtualMachine: VZVirtualMachine?
  var capturesSystemKeys: Bool?

  func makeNSView(context: Context) -> VZVirtualMachineView {
    let view = VZVirtualMachineView()
    view.virtualMachine = self.virtualMachine
    view.capturesSystemKeys = self.capturesSystemKeys ?? false
    DispatchQueue.main.async {
      self.view = view
    }
    return view
  }

  func updateNSView(_ view: VZVirtualMachineView, context: Context) {
    view.virtualMachine = self.virtualMachine
    view.capturesSystemKeys = self.capturesSystemKeys ?? false
  }
}

func runVMApp(instance: Instance) {
  _instance = instance
  VMApp.main()
}

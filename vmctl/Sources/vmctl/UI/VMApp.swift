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

  var body: some Scene {
    WindowGroup {
      VZVirtualMachineViewRepresentable(virtualMachine: _instance.vm, capturesSystemKeys: true)
        .frame(minWidth: 800, minHeight: 600, alignment: .center)
        .onReceive(_instance.onStop) { _ in
          NSApplication.shared.terminate(nil)
        }
    }
  }
}

func runVMApp(instance: Instance) {
  _instance = instance
  VMApp.main()
}

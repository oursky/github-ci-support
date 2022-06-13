import Combine
import HotKey
import SwiftUI
import Virtualization
import Vision

class KeyScriptRunner: ObservableObject {
  private var task: Task<(), Error>? = nil

  @Published private var current: (index: Int, instr: KeyScriptInstr)? = nil
  @Published var view: VZVirtualMachineView? = nil

  @Published var infoText: String?

  var instrText: String {
    guard let (index, instr) = self.current else {
      return "-"
    }
    var text = "#\(index): \(instr)"
    if let infoText = self.infoText {
      text += " | " + infoText
    }
    return text
  }

  init(script: [KeyScriptInstr]?) {
    self.task = Task.detached { @MainActor in
      let view: VZVirtualMachineView = await withCheckedContinuation { cont in
        var sub: AnyCancellable?
        sub = self.$view.sink { view in
          if let view = view {
            cont.resume(returning: view)
            sub?.cancel()
          }
        }
      }
      let window = view.window!
      var recognizer: TextRecognizer?

      for (index, instr) in (script ?? []).enumerated() {
        if Task.isCancelled {
          break
        }

        self.current = (index, instr)
        self.infoText = nil

        var events: [NSEvent] = []
        switch instr {
        case .keyDown(let key):
          events.append(keyEvent(key, down: true))

        case .keyUp(let key):
          events.append(keyEvent(key, down: false))

        case .keyPress(let keys):
          for key in keys {
            events.append(keyEvent(key, down: true))
          }
          for key in keys.reversed() {
            events.append(keyEvent(key, down: false))
          }

        case .text(let text):
          for char in text {
            let isUpper = char.isUppercase
            if isUpper {
              events.append(keyEvent(.shift, down: true))
            }
            guard let key = Key(string: char.lowercased()) else {
              self.infoText = "invalid key: \(char.lowercased())"
              return
            }
            events.append(keyEvent(key, down: true))
            events.append(keyEvent(key, down: false))
            if isUpper {
              events.append(keyEvent(.shift, down: false))
            }
          }

        case .sleep(let ms):
          try await sleep(ms: ms)

        case .waitFor(let text):
          recognizer = recognizer ?? TextRecognizer(view: view)
          while let haystack = try await recognizer?.recognizeText(),
            !haystack.lowercased().contains(text.lowercased())
          {
            self.infoText = haystack
            try await sleep(ms: 1000)
          }
        }

        for ev in events {
          window.postEvent(ev, atStart: false)
          try await sleep(ms: 33)
        }
      }
      self.current = nil
    }
  }
}

private func sleep(ms: UInt) async throws {
  try await Task.sleep(nanoseconds: UInt64(ms * 1_000_000))
}

private func keyEvent(_ key: Key, down: Bool) -> NSEvent {
  let cgEv = CGEvent(
    keyboardEventSource: nil, virtualKey: CGKeyCode(key.carbonKeyCode), keyDown: down)!
  let nsEv = NSEvent(cgEvent: cgEv)!
  return nsEv
}

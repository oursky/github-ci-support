import SwiftUI
import Vision

@MainActor
struct TextRecognizer {
  private let view: NSView
  private let repr: NSBitmapImageRep

  init(view: NSView) {
    self.view = view
    self.repr = view.bitmapImageRepForCachingDisplay(in: view.bounds)!
  }

  func recognizeText() async throws -> String {
    self.view.cacheDisplay(in: self.view.bounds, to: self.repr)

    let handler = VNImageRequestHandler(cgImage: repr.cgImage!)
    return try await withCheckedThrowingContinuation { cont in
      let request = VNRecognizeTextRequest { request, err in
        if let err = err {
          cont.resume(throwing: err)
          return
        }

        guard let observations = request.results as? [VNRecognizedTextObservation] else {
          cont.resume(returning: "")
          return
        }

        let string =
          observations
          .compactMap { $0.topCandidates(1).first?.string }
          .joined(separator: " ")
        cont.resume(returning: string)
      }
      request.recognitionLevel = .fast

      do {
        try handler.perform([request])
      } catch {
        cont.resume(throwing: error)
      }
    }
  }

}

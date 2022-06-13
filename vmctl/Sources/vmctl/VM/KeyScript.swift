import HotKey

enum KeyScriptInstr: Decodable {
  case keyDown(Key)
  case keyUp(Key)
  case keyPress([Key])
  case text(String)
  case sleep(ms: UInt)
  case waitFor(text: String)

  init(from decoder: Decoder) throws {
    let container = try decoder.singleValueContainer()
    let instr = try container.decode(String.self)
    let parts = instr.split(separator: ":", maxSplits: 1, omittingEmptySubsequences: true)
    if parts.count != 2 {
      throw DecodingError.dataCorruptedError(in: container, debugDescription: "invalid op format")
    }

    let op = parts.first!
    let data = parts.last!

    switch op {
    case "keyDown":
      guard let key = Key(string: String(data)) else {
        throw DecodingError.dataCorruptedError(
          in: container, debugDescription: "invalid key \(data)")
      }
      self = .keyDown(key)

    case "keyUp":
      guard let key = Key(string: String(data)) else {
        throw DecodingError.dataCorruptedError(
          in: container, debugDescription: "invalid key \(data)")
      }
      self = .keyUp(key)

    case "keyPress":
      let keys = try data.split(separator: ":").map { e -> Key in
        guard let key = Key(string: String(e)) else {
          throw DecodingError.dataCorruptedError(
            in: container, debugDescription: "invalid key \(data)")
        }
        return key
      }
      self = .keyPress(keys)

    case "sleep":
      guard let ms = UInt(data) else {
        throw DecodingError.dataCorruptedError(
          in: container, debugDescription: "invalid seconds \(data)")
      }
      self = .sleep(ms: ms)

    case "text":
      self = .text(String(data))

    case "waitFor":
      self = .waitFor(text: String(data))

    default:
      throw DecodingError.dataCorruptedError(in: container, debugDescription: "invalid op \(op)")
    }
  }
}

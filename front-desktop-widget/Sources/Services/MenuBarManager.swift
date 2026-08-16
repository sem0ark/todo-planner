import AppKit
import Combine

/// Manages the menu bar icon appearance based on widget state
class MenuBarManager: ObservableObject {
  static let shared = MenuBarManager()

  private var cancellables = Set<AnyCancellable>()
  weak var statusItem: NSStatusItem?

  private init() {}

  func configure(statusItem: NSStatusItem) {
    self.statusItem = statusItem
    updateIcon(state: .idle, category: nil)
  }

  /// Updates the menu bar icon based on current state
  func updateIcon(state: IconState, category: Category? = nil) {
    guard let button = statusItem?.button else { return }

    switch state {
    case .idle:
      button.image = NSImage(
        systemSymbolName: "checkmark.circle", accessibilityDescription: "Todo Planner - Idle")
    case .active:
      button.image = NSImage(
        systemSymbolName: "checkmark.circle.fill", accessibilityDescription: "Todo Planner - Active"
      )
    case .offSchedule:
      button.image = NSImage(
        systemSymbolName: "exclamationmark.circle.fill",
        accessibilityDescription: "Todo Planner - Off Schedule")
    case .confirmationNeeded:
      button.image = NSImage(
        systemSymbolName: "bell.fill", accessibilityDescription: "Todo Planner - Confirm")
    }

    // Optional: Tint the icon based on category color
    if let category = category, let color = NSColor(hexString: category.color) {
      button.contentTintColor = color
    } else {
      button.contentTintColor = nil
    }
  }

  enum IconState {
    case idle
    case active
    case offSchedule
    case confirmationNeeded
  }
}

extension NSColor {
  convenience init?(hexString: String) {
    let hex = hexString.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
    var int: UInt64 = 0
    Scanner(string: hex).scanHexInt64(&int)

    let r: UInt64
    let g: UInt64
    let b: UInt64
    switch hex.count {
    case 3:  // RGB (12-bit)
      (r, g, b) = ((int >> 8) * 17, (int >> 4 & 0xF) * 17, (int & 0xF) * 17)
    case 6:  // RGB (24-bit)
      (r, g, b) = (int >> 16, int >> 8 & 0xFF, int & 0xFF)
    case 8:  // ARGB (32-bit)
      (r, g, b) = (int >> 16 & 0xFF, int >> 8 & 0xFF, int & 0xFF)
    default:
      return nil
    }

    self.init(
      red: CGFloat(r) / 255,
      green: CGFloat(g) / 255,
      blue: CGFloat(b) / 255,
      alpha: 1.0
    )
  }
}

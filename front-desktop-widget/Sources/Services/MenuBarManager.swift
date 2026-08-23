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

    let image: NSImage?
    switch state {
    case .idle:
      image = NSImage(
        systemSymbolName: "checkmark.circle", accessibilityDescription: "Todo Planner - Idle")
    case .active:
      image = NSImage(
        systemSymbolName: "checkmark.circle.fill", accessibilityDescription: "Todo Planner - Active"
      )
    case .confirmationNeeded:
      image = NSImage(
        systemSymbolName: "bell.fill", accessibilityDescription: "Todo Planner - Confirm")
    }

    image?.isTemplate = true
    button.image = image
    button.contentTintColor = nil
  }

  enum IconState {
    case idle
    case active
    case confirmationNeeded
  }
}

import AppKit
import SwiftUI

@main
struct TodoPlannerWidgetApp: App {
  @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

  var body: some Scene {
    // Use Settings instead of WindowGroup to prevent Dock icon
    Settings {
      EmptyView()
    }
  }
}

class AppDelegate: NSObject, NSApplicationDelegate {
  var statusItem: NSStatusItem?
  var popover = NSPopover()
  var keyMonitor: Any?

  func applicationDidFinishLaunching(_ notification: Notification) {
    // Enforce singleton - check if another instance is already running
    if !ensureSingleInstance() {
      print("[APP] Another instance is already running. Activating existing instance...")
      NSApplication.shared.terminate(nil)
      return
    }

    print("[APP] Single instance check passed")
    print("[APP] Starting as menu bar app...")

    // Create the Menu Bar Item
    statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)

    if let button = statusItem?.button {
      button.action = #selector(togglePopover)
      button.target = self
    }

    // Configure menu bar manager
    if let item = statusItem {
      MenuBarManager.shared.configure(statusItem: item)
    }

    // Configure the Popover (The "Widget" View)
    popover.contentSize = NSSize(width: 320, height: 200)
    popover.behavior = .transient  // Closes when clicking outside
    popover.animates = true
    popover.contentViewController = NSHostingController(rootView: ContentView())

    // Register for URL events
    NSAppleEventManager.shared().setEventHandler(
      self,
      andSelector: #selector(handleGetURLEvent(_:withReplyEvent:)),
      forEventClass: AEEventClass(kInternetEventClass),
      andEventID: AEEventID(kAEGetURL)
    )

    // Install global keyboard event monitor (only when popover is showing)
    print("[APP] Menu bar app initialized successfully")

    // Listen for confirmation needed events
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handleConfirmationNeeded),
      name: .confirmationNeeded,
      object: nil
    )
    NotificationCenter.default.addObserver(
      self,
      selector: #selector(handlePomodoroCompleted),
      name: .pomodoroCompleted,
      object: nil
    )

    // Auto-open on launch
    DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) { [weak self] in
      self?.showPopover()
    }
  }

  @objc private func handleConfirmationNeeded() {
    print("[APP] Confirmation needed - auto-opening popover")
    showPopover()
  }

  @objc private func handlePomodoroCompleted() {
    print("[APP] Pomodoro completed - auto-opening popover")
    showPopover()
  }

  /// Ensures only one instance of the app is running
  /// Returns true if this is the only instance, false if another instance exists
  private func ensureSingleInstance() -> Bool {
    let bundleIdentifier = Bundle.main.bundleIdentifier ?? "com.todoplanner.widget"

    // Get all running applications with the same bundle identifier
    let runningApps = NSWorkspace.shared.runningApplications.filter {
      $0.bundleIdentifier == bundleIdentifier
    }

    // If more than one instance (including this one), another is already running
    if runningApps.count > 1 {
      // Try to activate the existing instance
      if let existingApp = runningApps.first(where: {
        $0.processIdentifier != ProcessInfo.processInfo.processIdentifier
      }) {
        existingApp.activate()
      }
      return false
    }

    return true
  }

  @objc func togglePopover(_ sender: AnyObject?) {
    if popover.isShown {
      closePopover()
    } else {
      showPopover()
    }
  }

  func showPopover() {
    guard let button = statusItem?.button else { return }

    // Activate the app to receive keyboard focus
    NSApp.activate(ignoringOtherApps: true)

    // Show the popover
    popover.show(relativeTo: button.bounds, of: button, preferredEdge: .minY)

    // Install keyboard monitor when popover opens
    installKeyboardMonitor()
    print("[APP] Popover opened")
  }

  func closePopover() {
    popover.performClose(nil)
    removeKeyboardMonitor()
    print("[APP] Popover closed")
  }

  private func installKeyboardMonitor() {
    // Remove existing monitor if any
    removeKeyboardMonitor()

    // Install new monitor
    keyMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
      // Only handle keys when popover is shown
      guard self?.popover.isShown == true else { return event }

      // Post notification for ContentView to handle
      NotificationCenter.default.post(name: .keyPressed, object: event)
      return event
    }
  }

  private func removeKeyboardMonitor() {
    if let monitor = keyMonitor {
      NSEvent.removeMonitor(monitor)
      keyMonitor = nil
    }
  }

  func applicationWillTerminate(_ notification: Notification) {
    removeKeyboardMonitor()
  }

  @objc func handleGetURLEvent(
    _ event: NSAppleEventDescriptor, withReplyEvent replyEvent: NSAppleEventDescriptor
  ) {
    if let urlString = event.paramDescriptor(forKeyword: AEKeyword(keyDirectObject))?.stringValue,
      let url = URL(string: urlString)
    {
      print("[DEEPLINK] Received URL: \(urlString)")

      // Show popover when deep link is received
      showPopover()

      // Handle the URL
      DeepLinkHandler.shared.handleURL(url)
    }
  }

  // Public method to show popover from anywhere (e.g., notifications, timers)
  func forceShowPopover() {
    showPopover()
  }
}

extension Notification.Name {
  static let keyPressed = Notification.Name("keyPressed")
}

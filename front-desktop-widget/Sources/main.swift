import SwiftUI

@main
struct TodoPlannerWidgetApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        WindowGroup {
            ContentView()
                .frame(width: 320, height: 200)
        }
        .windowStyle(.hiddenTitleBar)
        .windowResizability(.contentSize)
    }
}

class AppDelegate: NSObject, NSApplicationDelegate {
    var keyMonitor: Any?

    func applicationDidFinishLaunching(_ notification: Notification) {
        // Configure window to be always on top
        if let window = NSApplication.shared.windows.first {
            window.level = .floating
            window.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary]
            window.isMovableByWindowBackground = true
        }

        // Register for URL events
        NSAppleEventManager.shared().setEventHandler(
            self,
            andSelector: #selector(handleGetURLEvent(_:withReplyEvent:)),
            forEventClass: AEEventClass(kInternetEventClass),
            andEventID: AEEventID(kAEGetURL)
        )

        // Install global keyboard event monitor
        keyMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            return self?.handleKeyEvent(event) ?? event
        }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        return true
    }

    func applicationWillTerminate(_ notification: Notification) {
        if let monitor = keyMonitor {
            NSEvent.removeMonitor(monitor)
        }
    }

    @objc private func handleKeyEvent(_ event: NSEvent) -> NSEvent? {
        // Post notification with key event for ContentView to handle
        NotificationCenter.default.post(name: .keyPressed, object: event)
        return event
    }

    @objc func handleGetURLEvent(_ event: NSAppleEventDescriptor, withReplyEvent replyEvent: NSAppleEventDescriptor) {
        if let urlString = event.paramDescriptor(forKeyword: AEKeyword(keyDirectObject))?.stringValue,
           let url = URL(string: urlString) {
            // Bring app to front when URL is received
            NSApplication.shared.activate(ignoringOtherApps: true)

            // Show window if hidden
            if let window = NSApplication.shared.windows.first {
                window.makeKeyAndOrderFront(nil)
            }

            DeepLinkHandler.shared.handleURL(url)
        }
    }
}

extension Notification.Name {
    static let keyPressed = Notification.Name("keyPressed")
}

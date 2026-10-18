import Darwin
import Foundation

@MainActor
final class BackendManager {
  static let shared = BackendManager()

  private var process: Process?
  private var outputPipe: Pipe?
  private var errorPipe: Pipe?

  private init() {}

  func startBackend() {
    guard BuildConfig.storageMode == "local", process == nil else { return }
    guard let backendPath = Bundle.main.path(forResource: "todoplanner-backend", ofType: nil) else {
      WidgetLogger.error("Local backend executable is missing from the app bundle")
      return
    }

    let backendProcess = Process()
    backendProcess.executableURL = URL(fileURLWithPath: backendPath)

    var environment = ProcessInfo.processInfo.environment
    environment["APP_MODE"] = "local"
    environment["SERVE_FRONTEND"] = "true"
    environment["PORT"] = "8080"
    environment["EMBEDDED_PG_PORT"] = "54321"
    environment["EMBEDDED_PG_DATA_DIR"] = "~/.todoplanner"
    environment["TODO_PLANNER_PARENT_PID"] = String(ProcessInfo.processInfo.processIdentifier)
    backendProcess.environment = environment

    let standardOutput = Pipe()
    let standardError = Pipe()
    backendProcess.standardOutput = standardOutput
    backendProcess.standardError = standardError
    outputPipe = standardOutput
    errorPipe = standardError

    standardOutput.fileHandleForReading.readabilityHandler = { handle in
      let data = handle.availableData
      guard !data.isEmpty else { return }
      WidgetLogger.debug(
        "Local backend output",
        context: [
          "message": String(decoding: data, as: UTF8.self).trimmingCharacters(
            in: .whitespacesAndNewlines)
        ])
    }
    standardError.fileHandleForReading.readabilityHandler = { handle in
      let data = handle.availableData
      guard !data.isEmpty else { return }
      WidgetLogger.error(
        "Local backend error",
        context: [
          "message": String(decoding: data, as: UTF8.self).trimmingCharacters(
            in: .whitespacesAndNewlines)
        ])
    }

    backendProcess.terminationHandler = { [weak self] process in
      Task { @MainActor in
        WidgetLogger.debug(
          "Local backend terminated",
          context: ["status": String(process.terminationStatus)])
        self?.cleanup()
      }
    }

    do {
      try backendProcess.run()
      process = backendProcess
      WidgetLogger.debug("Local backend started")
    } catch {
      WidgetLogger.error(
        "Failed to start local backend", context: ["error": String(describing: error)])
      cleanup()
    }
  }

  func waitUntilReady() async -> Bool {
    guard BuildConfig.storageMode == "local" else { return true }

    guard let healthURL = URL(string: "http://127.0.0.1:8080/health") else { return false }
    for _ in 0..<100 {
      if process?.isRunning != true { return false }
      do {
        let (_, response) = try await URLSession.shared.data(from: healthURL)
        if (response as? HTTPURLResponse)?.statusCode == 200 { return true }
      } catch {
        // The process may still be starting PostgreSQL and applying migrations.
      }
      try? await Task.sleep(nanoseconds: 100_000_000)
    }
    return false
  }

  func stopBackend() {
    guard let backendProcess = process else { return }
    if backendProcess.isRunning {
      backendProcess.terminate()
      let shutdownDeadline = Date().addingTimeInterval(12)
      while backendProcess.isRunning && Date() < shutdownDeadline {
        Thread.sleep(forTimeInterval: 0.1)
      }
      if backendProcess.isRunning {
        kill(backendProcess.processIdentifier, SIGKILL)
        backendProcess.waitUntilExit()
      }
    }
    cleanup()
  }

  private func cleanup() {
    outputPipe?.fileHandleForReading.readabilityHandler = nil
    errorPipe?.fileHandleForReading.readabilityHandler = nil
    outputPipe = nil
    errorPipe = nil
    process = nil
  }
}

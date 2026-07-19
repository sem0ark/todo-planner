import Foundation
import SwiftUI

enum WidgetDisplayState {
    case confirmationPrompt // State 1: Planned block boundary reached
    case active             // State 2: On-schedule
    case offSchedule        // State 3: Distraction/off-plan
}

@MainActor
class WidgetState: ObservableObject {
    @Published var displayState: WidgetDisplayState = .active
    @Published var categories: [Category] = []
    @Published var currentDayRecord: DayRecord?
    @Published var currentCategory: Category?
    @Published var plannedCategory: Category?
    @Published var lastEventTime: Date = Date()
    @Published var progressPercentage: Double = 0.0
    @Published var showOffsetBar: Bool = false
    @Published var currentDuration: String = "0:00"
    @Published var isConfirmed: Bool = false
    @Published var offsetMinutes: Int = 0
    @Published var plannedDurationMinutes: Int = 0

    private var lastCheckedBlockId: Int?
    private var currentPlannedBlock: PlannedBlock?

    private let apiClient = APIClient.shared
    private var refreshTimer: Timer?
    private var offsetBarTimer: Timer?
    private var durationTimer: Timer?

    func initialize() async {
        print("[INIT] WidgetState: Starting initialization...")

        do {
            // Fetch categories
            print("[WIDGET] Fetching categories...")
            categories = try await apiClient.fetchCategories()
            print("[OK] Loaded \(categories.count) categories: \(categories.map { $0.name }.joined(separator: ", "))")

            // Fetch or create today's day record
            let today = todayString()
            print("[DATE] Today's date: \(today)")
            print("[IN] Fetching day records for \(today)...")
            let records = try await apiClient.fetchDayRecords(from: today, to: today)

            if let record = records.first {
                print("[OK] Found existing day record (id: \(record.id))")
                print("   - Snapshot ID: \(record.snapshotId?.description ?? "nil")")
                print("   - Snapshot blocks: \(record.snapshotBlocks.count)")
                print("   - Actual blocks: \(record.actualBlocks.count)")

                if record.snapshotId == nil {
                    print("[WARN] No template assigned for today - planned blocks will be empty")
                    print("[WARN] Please set up a weekly schedule via the web app")
                }

                currentDayRecord = record
                updateCurrentState()
            } else {
                // Create day record for today
                print("[CREATE] No day record found, creating new one...")
                currentDayRecord = try await apiClient.createDayRecord(date: today)
                print("[OK] Created day record (id: \(currentDayRecord?.id ?? -1))")

                if currentDayRecord?.snapshotId == nil {
                    print("[WARN] No template was assigned to the created day record")
                    print("[WARN] Please set up a weekly schedule via the web app")
                }
            }

            print("[DONE] Initialization complete!")
            print("[DEBUG] Final state:")
            print("   - currentCategory: \(currentCategory?.name ?? "nil")")
            print("   - plannedCategory: \(plannedCategory?.name ?? "nil")")
            print("   - displayState: \(displayState)")
            print("   - progressPercentage: \(progressPercentage)")
        } catch {
            print("[ERROR] Initialization error: \(error)")
            if let apiError = error as? APIError {
                switch apiError {
                case .invalidURL:
                    print("   → Invalid URL configuration")
                case .networkError(let underlyingError):
                    print("   → Network error: \(underlyingError.localizedDescription)")
                case .invalidResponse:
                    print("   → Invalid response from server")
                case .unauthorized:
                    print("   → Unauthorized - token may be invalid")
                case .serverError(let code, let message):
                    print("   → Server error \(code): \(message)")
                case .decodingError(let underlyingError):
                    print("   → Decoding error: \(underlyingError)")
                }
            }
        }
    }

    func confirmPlanned() async {
        print("[OK] WidgetState: Confirming planned activity...")

        guard let record = currentDayRecord else {
            print("[ERROR] No current day record")
            return
        }

        guard let planned = plannedCategory else {
            print("[ERROR] No planned category")
            return
        }

        print("   Day record ID: \(record.id)")
        print("   Planned category: \(planned.name)")

        do {
            let event = DayEvent(
                eventType: "confirmation",
                outgoingCategoryId: nil,
                incomingCategoryId: nil,
                occurredAt: Date()
            )

            print("[OUT] Posting confirmation event...")
            let response = try await apiClient.postDayEvents(
                dayRecordId: record.id,
                events: [event]
            )

            print("[OK] Received \(response.actualBlocks.count) actual blocks")
            // Update actual blocks
            updateDayRecordWithBlocks(response.actualBlocks)
            displayState = .active
            lastEventTime = Date()
            isConfirmed = true
            print("[OK] Confirmation complete, state: active")
        } catch {
            print("[ERROR] Confirm error: \(error)")
        }
    }

    func transitionToCategory(_ category: Category) async {
        print("[STATE] WidgetState: Transitioning to category '\(category.name)'...")

        guard let record = currentDayRecord else {
            print("[ERROR] No current day record")
            return
        }

        print("   From: \(currentCategory?.name ?? "none")")
        print("   To: \(category.name)")
        print("   Day record ID: \(record.id)")

        do {
            // If no current category (start of day), use the same category for both
            let outgoingId = currentCategory?.id ?? category.id

            let event = DayEvent(
                eventType: "transition",
                outgoingCategoryId: outgoingId,
                incomingCategoryId: category.id,
                occurredAt: Date()
            )

            print("[OUT] Posting transition event...")
            let response = try await apiClient.postDayEvents(
                dayRecordId: record.id,
                events: [event]
            )

            print("[OK] Received \(response.actualBlocks.count) actual blocks")
            updateDayRecordWithBlocks(response.actualBlocks)
            currentCategory = category
            lastEventTime = Date()
            isConfirmed = true
            offsetMinutes = 0 // Reset offset on new transition
            updateCurrentState()
            print("[OK] Transition complete")
        } catch {
            print("[ERROR] Transition error: \(error)")
        }
    }

    func syncToPlan() async {
        guard let planned = plannedCategory else { return }
        await transitionToCategory(planned)
    }

    func adjustOffset(minutes: Int) async {
        // Increase offset counter (note: minutes parameter is already positive)
        offsetMinutes += minutes

        // Adjust last event timestamp backwards
        lastEventTime = lastEventTime.addingTimeInterval(TimeInterval(-minutes * 60))

        print("[OFFSET] Adjusted by +\(minutes)m, total offset: \(offsetMinutes)m")
        // TODO: Send retroactive edit to API when backend supports it
    }

    private func updateCurrentState() {
        guard let record = currentDayRecord else { return }

        let now = Date()
        let currentPlanned = getCurrentPlannedBlock(at: now, from: record.snapshotBlocks)
        let currentActual = getCurrentActualBlock(at: now, from: record.actualBlocks)

        // Always set planned category and progress if there's a current planned block
        if let planned = currentPlanned {
            currentPlannedBlock = planned
            plannedDurationMinutes = planned.durationMinutes
            plannedCategory = categories.first { $0.id == planned.categoryId }
            progressPercentage = calculateProgress(for: planned, at: now)

            print("[STATE] Current planned block: \(plannedCategory?.name ?? "unknown")")
            print("[STATE] Progress: \(Int(progressPercentage * 100))%")

            // Check if we're within 1 minute of block start (confirmation window)
            let isAtBoundary = isWithinConfirmationWindow(for: planned, at: now)

            if lastCheckedBlockId != planned.id {
                // New block boundary detected
                lastCheckedBlockId = planned.id
                isConfirmed = false

                if isAtBoundary {
                    print("[STATE] At block boundary - showing confirmation prompt")
                    displayState = .confirmationPrompt
                }
            }
        } else {
            print("[WARN] No current planned block found")
        }

        // Set current category from actual block, or default to planned on startup
        if let actual = currentActual, let catId = actual.categoryId {
            currentCategory = categories.first { $0.id == catId }
            print("[STATE] Current category from actual block: \(currentCategory?.name ?? "unknown")")
        } else if currentCategory == nil, let planned = plannedCategory {
            // On startup with no actual blocks yet, show planned category
            // This will be overridden once user makes their first manual selection
            currentCategory = planned
            print("[STATE] No actual block yet, showing planned category: \(planned.name)")
        }

        // Determine display state (only if not in confirmation mode)
        if displayState != .confirmationPrompt {
            if let current = currentCategory, let planned = plannedCategory {
                if current.id == planned.id {
                    displayState = .active
                    showOffsetBar = false
                    print("[STATE] Display state: .active (on schedule)")
                } else {
                    let wasOffSchedule = displayState == .offSchedule
                    displayState = .offSchedule
                    print("[STATE] Display state: .offSchedule")

                    // Start offset bar timer on new off-schedule transition
                    if !wasOffSchedule {
                        showOffsetBar = true
                        startOffsetBarTimer()
                    }
                }
            } else {
                print("[WARN] Cannot determine display state - missing current or planned category")
            }
        }
    }

    private func isWithinConfirmationWindow(for block: PlannedBlock, at time: Date) -> Bool {
        let calendar = Calendar.current

        // Parse time - handle both "HH:mm:ss" and "HH:mm:ss.SSSSSS" formats
        let timeString = block.startTime.components(separatedBy: ".").first ?? block.startTime
        let timeParts = timeString.split(separator: ":").compactMap { Int($0) }
        guard timeParts.count == 3 else { return false }

        let blockStartSeconds = timeParts[0] * 3600 + timeParts[1] * 60 + timeParts[2]

        let currentComponents = calendar.dateComponents([.hour, .minute, .second], from: time)
        let currentSeconds = (currentComponents.hour ?? 0) * 3600 + (currentComponents.minute ?? 0) * 60 + (currentComponents.second ?? 0)

        let elapsed = currentSeconds - blockStartSeconds

        // Within 1 minute of start (0-60 seconds)
        return elapsed >= 0 && elapsed < 60 && !isConfirmed
    }

    private func startOffsetBarTimer() {
        offsetBarTimer?.invalidate()
        offsetBarTimer = Timer.scheduledTimer(withTimeInterval: 120, repeats: false) { [weak self] _ in
            Task { @MainActor in
                self?.showOffsetBar = false
            }
        }
    }

    private func calculateProgress(for block: PlannedBlock, at time: Date) -> Double {
        let calendar = Calendar.current

        // Parse time - handle both "HH:mm:ss" and "HH:mm:ss.SSSSSS" formats
        let timeString = block.startTime.components(separatedBy: ".").first ?? block.startTime
        let timeParts = timeString.split(separator: ":").compactMap { Int($0) }
        guard timeParts.count == 3 else { return 0 }

        let blockStartSeconds = timeParts[0] * 3600 + timeParts[1] * 60 + timeParts[2]

        let currentComponents = calendar.dateComponents([.hour, .minute, .second], from: time)
        let currentSeconds = (currentComponents.hour ?? 0) * 3600 + (currentComponents.minute ?? 0) * 60 + (currentComponents.second ?? 0)
        let blockDurationSeconds = block.durationMinutes * 60

        let elapsed = max(0, currentSeconds - blockStartSeconds)
        return min(1.0, Double(elapsed) / Double(blockDurationSeconds))
    }

    private func getCurrentPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
        let calendar = Calendar.current

        for block in blocks {
            // Parse time - handle both "HH:mm:ss" and "HH:mm:ss.SSSSSS" formats
            let timeString = block.startTime.components(separatedBy: ".").first ?? block.startTime
            let timeParts = timeString.split(separator: ":").compactMap { Int($0) }
            guard timeParts.count == 3 else { continue }

            let blockStartSeconds = timeParts[0] * 3600 + timeParts[1] * 60 + timeParts[2]

            let currentComponents = calendar.dateComponents([.hour, .minute, .second], from: time)
            let currentSeconds = (currentComponents.hour ?? 0) * 3600 + (currentComponents.minute ?? 0) * 60 + (currentComponents.second ?? 0)
            let blockEndSeconds = blockStartSeconds + (block.durationMinutes * 60)

            if currentSeconds >= blockStartSeconds && currentSeconds < blockEndSeconds {
                return block
            }
        }

        return nil
    }

    private func getCurrentActualBlock(at time: Date, from blocks: [ActualBlock]) -> ActualBlock? {
        let calendar = Calendar.current

        for block in blocks {
            // Parse time - handle both "HH:mm:ss" and "HH:mm:ss.SSSSSS" formats
            let timeString = block.startTime.components(separatedBy: ".").first ?? block.startTime
            let timeParts = timeString.split(separator: ":").compactMap { Int($0) }
            guard timeParts.count == 3 else { continue }

            let blockStartSeconds = timeParts[0] * 3600 + timeParts[1] * 60 + timeParts[2]

            let currentComponents = calendar.dateComponents([.hour, .minute, .second], from: time)
            let currentSeconds = (currentComponents.hour ?? 0) * 3600 + (currentComponents.minute ?? 0) * 60 + (currentComponents.second ?? 0)
            let blockEndSeconds = blockStartSeconds + (block.durationMinutes * 60)

            if currentSeconds >= blockStartSeconds && currentSeconds < blockEndSeconds {
                return block
            }
        }

        return nil
    }

    private func updateDayRecordWithBlocks(_ blocks: [ActualBlock]) {
        guard let record = currentDayRecord else { return }
        // Create updated record with new actual blocks
        currentDayRecord = DayRecord(
            id: record.id,
            snapshotId: record.snapshotId,
            calendarDate: record.calendarDate,
            reviewStatus: record.reviewStatus,
            snapshotBlocks: record.snapshotBlocks,
            actualBlocks: blocks,
            createdAt: record.createdAt,
            updatedAt: Date()
        )
    }

    private func todayString() -> String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return formatter.string(from: Date())
    }

    func startPeriodicRefresh() {
        refreshTimer = Timer.scheduledTimer(withTimeInterval: 60, repeats: true) { [weak self] _ in
            Task {
                await self?.updateCurrentState()
            }
        }

        // Start duration timer (updates every second)
        durationTimer = Timer.scheduledTimer(withTimeInterval: 1.0, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.updateDuration()
            }
        }
    }

    func stopPeriodicRefresh() {
        refreshTimer?.invalidate()
        refreshTimer = nil
        durationTimer?.invalidate()
        durationTimer = nil
    }

    private func updateDuration() {
        let elapsed = Date().timeIntervalSince(lastEventTime)
        let minutes = Int(elapsed) / 60
        let seconds = Int(elapsed) % 60
        currentDuration = String(format: "%d:%02d", minutes, seconds)
    }
}

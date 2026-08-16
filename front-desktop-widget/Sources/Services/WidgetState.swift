import Combine
import Foundation
import SwiftUI

/*
stateDiagram-v2
    direction TB

    state "INITIALIZING" as Init
    state "STATE 1: CONFIRMATION_PROMPT" as S1
    state "STATE 2: ACTIVE (ON-SCHEDULE)" as S2
    state "STATE 3: OFF-SCHEDULE" as S3

    [*] --> Init : App Launch
    
    Init --> S2 : Record Found / On Plan
    Init --> S3 : Record Found / Off Plan
    Init --> S1 : Boundary Reached during Init

    %% State 1 Logic
    state S1 {
        [*] --> PulsingUI
        PulsingUI --> PulsingUI : Timer Tick (Breathing)
    }
    S1 --> S2 : Space / Click Left (Confirm Plan)
    S1 --> S3 : Click Category [N] (Unplanned Start)

    %% State 2 Logic (Including Pomodoro)
    state S2 {
        [*] --> StandardActive
        
        state "POMODORO_MODE" as Pomo {
            state "Work Phase" as PomoWork
            state "Rest Phase" as PomoRest
            
            [*] --> PomoWork
            PomoWork --> PomoRest : Timer End / Space (if >100%)
            PomoRest --> PomoWork : Space / Auto-Skip (1.5x duration)
        }
        
        StandardActive --> Pomo : Category.hasPomodoro == true
        Pomo --> StandardActive : Category.hasPomodoro == false
    }

    S2 --> S3 : Click Category [N] (Distraction logged)
    S2 --> S1 : Block Boundary Reached (New Plan)

    %% State 3 Logic
    state S3 {
        [*] --> SplitView
        
        state "OFFSET_WINDOW" as Offset {
            [*] --> Visible : 120s Timer Start
            Visible --> Hidden : Timer Expired
            Visible --> Visible : [ or ] Key (Nudge -5m)
        }
        
        SplitView --> SplitView : [ or ] Key (Update Timestamp)
    }

    S3 --> S2 : Enter / Click Bottom (Sync to Plan)
    S3 --> S3 : Click Category [N] (New Distraction)
    S3 --> S1 : Block Boundary Reached (New Plan)

    %% Global Transitions
    S2 --> S2 : Space (Confirmation Pulse)
    S3 --> Init : Cmd+Z (Undo to previous state)
*/

extension Notification.Name {
  static let confirmationNeeded = Notification.Name("confirmationNeeded")
}

enum PomodoroPhase {
  case work
  case rest
}

struct PomodoroState {
  var phase: PomodoroPhase
  var elapsed: Int  // seconds
}

enum WidgetStateIdentity {
  case initializing
  case confirmationPrompt
  case active
  case offSchedule
}

typealias WidgetDisplayState = WidgetStateIdentity

struct WidgetContext {
  var categories: [Category] = []
  var currentDayRecord: DayRecord?
  var currentCategory: Category?
  var plannedCategory: Category?
  var lastEventTime = Date()
  var isConfirmed = false
  var pomodoroPhase: PomodoroPhase = .work
  var pomodoroElapsed = 0
  var offsetMinutes = 0
  var offsetExpiry: Date?
}

enum WidgetAction {
  case initialize
  case confirm
  case selectCategory(Category)
  case syncToPlan
  case adjustOffset(Int)
  case spacePressed
}

enum DayEventType: String {
  case confirmation
  case transition
}

@MainActor
final class WidgetState: ObservableObject {
  @Published private(set) var stateIdentity: WidgetStateIdentity = .initializing
  @Published private(set) var context = WidgetContext()

  // Compatibility projections used by the existing SwiftUI views.
  @Published var displayState: WidgetDisplayState = .initializing
  @Published var categories: [Category] = []
  @Published var currentDayRecord: DayRecord?
  @Published var currentCategory: Category?
  @Published var plannedCategory: Category?
  @Published var lastEventTime = Date()
  @Published var progressPercentage = 0.0
  @Published var showOffsetBar = false
  @Published var currentDuration = "0:00"
  @Published var isConfirmed = false
  @Published var offsetMinutes = 0
  @Published var plannedDurationMinutes = 0
  @Published var pomodoroState: PomodoroState?
  @Published var pomodoroProgress = 0.0

  private let apiClient = APIClient.shared
  private var timer: AnyCancellable?
  private var lastCheckedBlockId: Int?
  private var currentPlannedBlock: PlannedBlock?

  var pomodoroActive: Bool {
    stateIdentity == .active && context.isConfirmed
      && context.currentCategory?.hasPomodoroEnabled == true
  }

  var pomodoroPulsing: Bool { pomodoroActive && pomodoroProgress >= 1.0 }

  init() {
    startGlobalTimer()
  }

  // MARK: - External Actions

  func handleAction(_ action: WidgetAction) async {
    print(
      "[ACTION] Received: \(actionDescription(action)); state=\(stateDescription(stateIdentity))")

    switch action {
    case .initialize:
      await performInitialization()
    case .confirm:
      await transition(to: .active, eventType: .confirmation)
    case .selectCategory(let category):
      let target: WidgetStateIdentity =
        category.id == context.plannedCategory?.id ? .active : .offSchedule
      await transition(to: target, category: category, eventType: .transition)
    case .syncToPlan:
      guard let planned = context.plannedCategory else {
        print("[ACTION] syncToPlan ignored: no planned category")
        return
      }
      await transition(to: .active, category: planned, eventType: .transition)
    case .adjustOffset(let minutes):
      await applyOffset(minutes)
    case .spacePressed:
      await handleSpaceKeyAction()
    }

    print(
      "[ACTION] Completed: \(actionDescription(action)); state=\(stateDescription(stateIdentity))")
  }

  // MARK: - Initialization

  private func performInitialization() async {
    print("[INIT] WidgetState: Starting initialization")

    do {
      let loadedCategories = try await apiClient.fetchCategories()
      let today = todayString()
      let records = try await apiClient.fetchDayRecords(from: today, to: today)
      let record = records.first ?? try await apiClient.createDayRecord(date: today)

      context.categories = loadedCategories
      context.currentDayRecord = record
      categories = loadedCategories
      currentDayRecord = record
      setState(.active)
      updateCurrentState()
      print("[INIT] Complete: categories=\(loadedCategories.count), record=\(record.id)")
    } catch {
      print("[ERROR] Initialization error: \(error)")
    }
  }

  func initialize() async { await handleAction(.initialize) }

  // MARK: - State Transitions

  private func transition(
    to target: WidgetStateIdentity, category: Category? = nil, eventType: DayEventType? = nil
  ) async {
    let previous = stateIdentity
    let eventTime = Date()

    if let category {
      context.currentCategory = category
      currentCategory = category
      context.pomodoroPhase = .work
      context.pomodoroElapsed = 0
      pomodoroState = nil
      pomodoroProgress = 0
    }
    context.lastEventTime = eventTime
    context.isConfirmed = true
    lastEventTime = eventTime
    isConfirmed = true
    setState(target)

    if target == .offSchedule {
      context.offsetExpiry = eventTime.addingTimeInterval(120)
      context.offsetMinutes = 0
      offsetMinutes = 0
      showOffsetBar = true
    } else {
      context.offsetExpiry = nil
      showOffsetBar = false
    }

    print(
      "[STATE] Transition \(stateDescription(previous)) -> \(stateDescription(target))"
        + " category=\(context.currentCategory?.name ?? "none") event=\(eventType?.rawValue ?? "none")"
    )

    if let eventType, let record = context.currentDayRecord {
      let event = DayEvent(
        eventType: eventType.rawValue, outgoingCategoryId: nil, incomingCategoryId: category?.id,
        occurredAt: eventTime)
      do {
        let response = try await apiClient.postDayEvents(dayRecordId: record.id, events: [event])
        updateDayRecordWithBlocks(response.actualBlocks)
        print(
          "[SYNC] Event persisted: \(eventType.rawValue), blocks=\(response.actualBlocks.count)")
      } catch {
        print("[SYNC] Failed to persist \(eventType.rawValue): \(error)")
      }
    }

    updateCurrentState()
  }

  private func setState(_ state: WidgetStateIdentity) {
    stateIdentity = state
    displayState = state
    updateMenuBarIcon()
  }

  private func stateDescription(_ state: WidgetStateIdentity) -> String {
    switch state {
    case .initializing: "initializing"
    case .confirmationPrompt: "confirmationPrompt"
    case .active: "active"
    case .offSchedule: "offSchedule"
    }
  }

  private func actionDescription(_ action: WidgetAction) -> String {
    switch action {
    case .initialize: "initialize"
    case .confirm: "confirm"
    case .selectCategory(let category): "selectCategory(\(category.name))"
    case .syncToPlan: "syncToPlan"
    case .adjustOffset(let minutes): "adjustOffset(\(minutes)m)"
    case .spacePressed: "spacePressed"
    }
  }

  // MARK: - Ticker

  private func startGlobalTimer() {
    timer = Timer.publish(every: 1.0, on: .main, in: .common)
      .autoconnect()
      .sink { [weak self] _ in self?.tick() }
  }

  private func tick() {
    let now = Date()

    if stateIdentity != .confirmationPrompt, let block = currentPlannedBlock,
      isWithinConfirmationWindow(for: block, at: now), lastCheckedBlockId != block.id
    {
      lastCheckedBlockId = block.id
      context.isConfirmed = false
      isConfirmed = false
      setState(.confirmationPrompt)
      print("[STATE] Boundary reached: active -> confirmationPrompt (block=\(block.id))")
      NotificationCenter.default.post(name: .confirmationNeeded, object: nil)
    }

    if let expiry = context.offsetExpiry, now > expiry {
      context.offsetExpiry = nil
      showOffsetBar = false
      print("[STATE] Offset window expired")
    }

    updateDerivedUI()
    if pomodoroActive { updatePomodoroLogic() }
  }

  func startPeriodicRefresh() {
    if timer == nil { startGlobalTimer() }
  }

  func stopPeriodicRefresh() {
    timer?.cancel()
    timer = nil
  }

  // MARK: - Compatibility API

  func confirmPlanned() async { await handleAction(.confirm) }
  func transitionToCategory(_ category: Category) async {
    await handleAction(.selectCategory(category))
  }
  func syncToPlan() async { await handleAction(.syncToPlan) }
  func adjustOffset(minutes: Int) async { await handleAction(.adjustOffset(minutes)) }
  func handleSpaceKey() async { await handleAction(.spacePressed) }

  // MARK: - Derived UI

  private func updateCurrentState() {
    guard let record = context.currentDayRecord else { return }
    let now = Date()
    let currentPlanned = getCurrentPlannedBlock(at: now, from: record.snapshotBlocks)
    let planned = currentPlanned ?? getNextPlannedBlock(at: now, from: record.snapshotBlocks)
    currentPlannedBlock = planned

    if let planned {
      plannedDurationMinutes = planned.durationMinutes
      context.plannedCategory = context.categories.first { $0.id == planned.categoryId }
      plannedCategory = context.plannedCategory
      progressPercentage = currentPlanned.map { calculateProgress(for: $0, at: now) } ?? 0
    }

    if let actual = getCurrentActualBlock(at: now, from: record.actualBlocks),
      let categoryId = actual.categoryId
    {
      context.currentCategory = context.categories.first { $0.id == categoryId }
    } else if context.currentCategory == nil {
      context.currentCategory = context.plannedCategory
    }
    currentCategory = context.currentCategory

    if stateIdentity != .confirmationPrompt, let current = context.currentCategory,
      let plan = context.plannedCategory
    {
      setState(current.id == plan.id ? .active : .offSchedule)
    }
    updateDerivedUI()
  }

  private func updateDerivedUI() {
    let elapsed = max(0, Int(Date().timeIntervalSince(context.lastEventTime)))
    currentDuration = String(format: "%d:%02d", elapsed / 60, elapsed % 60)
    lastEventTime = context.lastEventTime
    isConfirmed = context.isConfirmed
    offsetMinutes = context.offsetMinutes
    categories = context.categories
    currentDayRecord = context.currentDayRecord
    plannedCategory = context.plannedCategory
    currentCategory = context.currentCategory
  }

  private func handleSpaceKeyAction() async {
    switch stateIdentity {
    case .confirmationPrompt:
      await handleAction(.confirm)
    case .active:
      guard pomodoroActive, let state = pomodoroState, state.phase == .work, pomodoroProgress >= 1
      else { return }
      context.pomodoroPhase = .rest
      context.pomodoroElapsed = 0
      pomodoroState = PomodoroState(phase: .rest, elapsed: 0)
      pomodoroProgress = 0
      print("[POMODORO] Work complete -> rest")
    case .initializing, .offSchedule:
      break
    }
  }

  private func applyOffset(_ minutes: Int) async {
    guard let record = context.currentDayRecord, let current = context.currentCategory else {
      return
    }
    let retroactiveTime = context.lastEventTime.addingTimeInterval(TimeInterval(-minutes * 60))
    context.offsetMinutes += minutes
    context.lastEventTime = retroactiveTime
    lastEventTime = retroactiveTime
    offsetMinutes = context.offsetMinutes

    let event = DayEvent(
      eventType: DayEventType.transition.rawValue, outgoingCategoryId: current.id,
      incomingCategoryId: current.id, occurredAt: retroactiveTime)
    do {
      let response = try await apiClient.postDayEvents(dayRecordId: record.id, events: [event])
      updateDayRecordWithBlocks(response.actualBlocks)
      print("[OFFSET] Applied \(minutes)m, total=\(context.offsetMinutes)m")
    } catch {
      print("[OFFSET] Sync failed: \(error)")
    }
    updateCurrentState()
  }

  // MARK: - Pomodoro

  private func updatePomodoroLogic() {
    guard let config = context.currentCategory?.pomodoroConfig else { return }
    context.pomodoroElapsed += 1
    pomodoroState = PomodoroState(phase: context.pomodoroPhase, elapsed: context.pomodoroElapsed)
    let duration = context.pomodoroPhase == .work ? config.workDuration : config.restDuration
    pomodoroProgress = duration > 0 ? min(Double(context.pomodoroElapsed) / Double(duration), 1) : 0
    if context.pomodoroPhase == .rest
      && context.pomodoroElapsed > Int(Double(config.restDuration) * 1.5)
    {
      context.pomodoroPhase = .work
      context.pomodoroElapsed = 0
      pomodoroState = PomodoroState(phase: .work, elapsed: 0)
      pomodoroProgress = 0
    }
  }

  // MARK: - Helpers

  private func updateDayRecordWithBlocks(_ blocks: [ActualBlock]) {
    guard let record = context.currentDayRecord else { return }
    let updated = DayRecord(
      id: record.id, snapshotId: record.snapshotId, calendarDate: record.calendarDate,
      reviewStatus: record.reviewStatus, snapshotBlocks: record.snapshotBlocks,
      actualBlocks: blocks, createdAt: record.createdAt, updatedAt: Date())
    context.currentDayRecord = updated
    currentDayRecord = updated
  }

  private func updateMenuBarIcon() {
    let icon: MenuBarManager.IconState =
      switch stateIdentity {
      case .initializing: .active
      case .confirmationPrompt: .confirmationNeeded
      case .active: .active
      case .offSchedule: .offSchedule
      }
    MenuBarManager.shared.updateIcon(state: icon, category: context.currentCategory)
  }

  private func isWithinConfirmationWindow(for block: PlannedBlock, at time: Date) -> Bool {
    let elapsed = secondsSinceStart(of: time) - seconds(from: block.startTime)
    return elapsed >= 0 && elapsed < 60 && !context.isConfirmed
  }

  private func calculateProgress(for block: PlannedBlock, at time: Date) -> Double {
    let elapsed = max(0, secondsSinceStart(of: time) - seconds(from: block.startTime))
    return min(1, Double(elapsed) / Double(max(1, block.durationMinutes * 60)))
  }

  private func getCurrentPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    let current = secondsSinceStart(of: time)
    return blocks.first { block in
      let begin = seconds(from: block.startTime)
      return current >= begin && current < begin + block.durationMinutes * 60
    }
  }

  private func getNextPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    let current = secondsSinceStart(of: time)
    return blocks.filter { seconds(from: $0.startTime) > current }
      .min { seconds(from: $0.startTime) < seconds(from: $1.startTime) } ?? blocks.first
  }

  private func getCurrentActualBlock(at time: Date, from blocks: [ActualBlock]) -> ActualBlock? {
    let current = secondsSinceStart(of: time)
    return blocks.first { block in
      let begin = seconds(from: block.startTime)
      return current >= begin && current < begin + block.durationMinutes * 60
    }
  }

  private func secondsSinceStart(of date: Date) -> Int {
    let components = Calendar.current.dateComponents([.hour, .minute, .second], from: date)
    return (components.hour ?? 0) * 3600 + (components.minute ?? 0) * 60 + (components.second ?? 0)
  }

  private func seconds(from time: String) -> Int {
    let parts = time.split(separator: ".").first?.split(separator: ":").compactMap { Int($0) } ?? []
    guard parts.count == 3 else { return 0 }
    return parts[0] * 3600 + parts[1] * 60 + parts[2]
  }

  private func todayString() -> String {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    return formatter.string(from: Date())
  }
}

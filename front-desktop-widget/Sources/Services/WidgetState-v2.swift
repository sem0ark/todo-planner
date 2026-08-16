import Combine
import Foundation
import Observation
import SwiftUI

// MARK: - Core Type Definitions

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
  var lastCheckedBlockId: Int?
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

// MARK: - State Protocol & Result

/// The atomic output of a state operation.
struct StateResult {
  let nextState: WidgetStateLogic
  let updatedContext: WidgetContext
}

/// The blueprint for all widget state logic modules.
@MainActor
protocol WidgetStateLogic {
  var identity: WidgetStateIdentity { get }

  /// Processes user intents (e.g., button clicks, key presses).
  func handle(
    action: WidgetAction,
    context: WidgetContext,
    repository: TodoPlannerRepository
  ) async -> StateResult

  /// Processes temporal events (e.g., 1s heartbeat, boundary checks).
  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult
}

// MARK: - Supporting Utilities

struct TimeLogic {
  static func getCurrentPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    let current = secondsSinceStartOfDay(for: time)
    return blocks.first { block in
      let begin = parseSeconds(from: block.startTime)
      return current >= begin && current < begin + block.durationMinutes * 60
    }
  }

  static func getNextPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    let current = secondsSinceStartOfDay(for: time)
    return blocks.filter { parseSeconds(from: $0.startTime) > current }
      .min { parseSeconds(from: $0.startTime) < parseSeconds(from: $1.startTime) } ?? blocks.first
  }

  static func getCurrentActualBlock(at time: Date, from blocks: [ActualBlock]) -> ActualBlock? {
    let current = secondsSinceStartOfDay(for: time)
    return blocks.last { block in
      let begin = parseSeconds(from: block.startTime)
      let isOpenEnded = block.durationMinutes <= 0
      return current >= begin && (isOpenEnded || current < begin + block.durationMinutes * 60)
    }
  }

  static func calculateProgress(for block: PlannedBlock, at time: Date) -> Double {
    let elapsed = max(0, secondsSinceStartOfDay(for: time) - parseSeconds(from: block.startTime))
    return min(1, Double(elapsed) / Double(max(1, block.durationMinutes * 60)))
  }

  static func isWithinConfirmationWindow(for block: PlannedBlock, at time: Date) -> Bool {
    let elapsed = secondsSinceStartOfDay(for: time) - parseSeconds(from: block.startTime)
    return elapsed >= 0 && elapsed < 60
  }

  static func secondsSinceStartOfDay(for date: Date) -> Int {
    let components = Calendar.current.dateComponents([.hour, .minute, .second], from: date)
    return (components.hour ?? 0) * 3600 + (components.minute ?? 0) * 60
      + (components.second ?? 0)
  }

  static func parseSeconds(from timeString: String) -> Int {
    let parts =
      timeString.split(separator: ".").first?.split(separator: ":").compactMap { Int($0) }
      ?? []
    guard parts.count == 3 else { return 0 }
    return parts[0] * 3600 + parts[1] * 60 + parts[2]
  }
}

extension DateFormatter {
  static let yyyyMMdd: DateFormatter = {
    let formatter = DateFormatter()
    formatter.dateFormat = "yyyy-MM-dd"
    return formatter
  }()
}

// MARK: - Context Extensions (Centralized Logic)

extension WidgetContext {
  /// Updates the day record with new actual blocks
  mutating func updateBlocks(_ blocks: [ActualBlock]) {
    guard let record = currentDayRecord else { return }
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

  /// Updates lastCheckedBlockId for boundary detection
  mutating func markBlockAsChecked(_ blockId: Int) {
    lastCheckedBlockId = blockId
  }

  /// Performs a category transition with all side effects
  @MainActor
  mutating func transitionTo(
    category: Category, isPlanned: Bool, repo: TodoPlannerRepository
  ) async -> StateResult {
    currentCategory = category
    lastEventTime = Date()
    isConfirmed = true
    pomodoroPhase = .work
    pomodoroElapsed = 0

    if isPlanned {
      offsetExpiry = nil
      offsetMinutes = 0
    } else {
      offsetExpiry = Date().addingTimeInterval(120)
      offsetMinutes = 0
    }

    let blocks = await logEvent(
      type: .transition, category: category, occurredAt: nil, recordId: currentDayRecord?.id,
      repo: repo)
    updateBlocks(blocks)

    let nextState: WidgetStateLogic
    let stateName: String
    if isPlanned {
      nextState = ActiveState()
      stateName = "active"
    } else {
      nextState = OffScheduleState()
      stateName = "offSchedule"
    }
    print("[STATE] Transition -> \(stateName) category=\(category.name)")
    return StateResult(nextState: nextState, updatedContext: self)
  }

  /// Confirms the current plan
  @MainActor
  mutating func confirm(repo: TodoPlannerRepository) async -> StateResult {
    isConfirmed = true
    lastEventTime = Date()

    let blocks = await logEvent(
      type: .confirmation, category: plannedCategory, occurredAt: nil,
      recordId: currentDayRecord?.id, repo: repo)
    updateBlocks(blocks)

    print("[CONFIRM] Plan validated")
    let nextState = ActiveState()
    return StateResult(nextState: nextState, updatedContext: self)
  }

  /// Updates Pomodoro state (called every second in active mode)
  mutating func tickPomodoro() {
    guard let config = currentCategory?.pomodoroConfig else { return }

    pomodoroElapsed += 1
    let limit = pomodoroPhase == .work ? config.workDuration : config.restDuration

    // Auto-reset if rest phase exceeds 1.5x duration
    if pomodoroPhase == .rest && pomodoroElapsed > Int(Double(limit) * 1.5) {
      pomodoroPhase = .work
      pomodoroElapsed = 0
      print("[POMODORO] Auto-reset rest -> work")
    }
  }

  /// Toggles Pomodoro phase (space key in active state)
  mutating func togglePomodoro() {
    guard let config = currentCategory?.pomodoroConfig else { return }

    if pomodoroPhase == .work && pomodoroElapsed >= config.workDuration {
      pomodoroPhase = .rest
      pomodoroElapsed = 0
      print("[POMODORO] Work complete -> rest")
    } else if pomodoroPhase == .rest {
      pomodoroPhase = .work
      pomodoroElapsed = 0
      print("[POMODORO] Rest skipped -> work")
    }
  }
}

/// Unified event logging helper
@MainActor
func logEvent(
  type: DayEventType,
  category: Category?,
  occurredAt: Date?,
  recordId: Int?,
  repo: TodoPlannerRepository
) async -> [ActualBlock] {
  guard let recordId = recordId else { return [] }

  let event = DayEvent(
    eventType: type.rawValue,
    outgoingCategoryId: type == .transition ? category?.id : nil,
    incomingCategoryId: category?.id,
    occurredAt: occurredAt ?? Date()
  )

  do {
    let response = try await repo.submitEvents(dayRecordId: recordId, events: [event])
    print("[SYNC] Event persisted: \(type.rawValue), blocks=\(response.actualBlocks.count)")
    return response.actualBlocks
  } catch {
    print("[SYNC ERROR] Failed to log \(type): \(error.localizedDescription)")
    return []
  }
}

// MARK: - Concrete State: Initializing

final class InitializingState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .initializing

  func handle(
    action: WidgetAction,
    context: WidgetContext,
    repository: TodoPlannerRepository
  ) async -> StateResult {
    guard case .initialize = action else {
      return StateResult(nextState: self, updatedContext: context)
    }

    var ctx = context

    do {
      print("[INIT] WidgetState: Starting initialization")

      ctx.categories = try await repository.fetchCategories()

      let today = DateFormatter.yyyyMMdd.string(from: Date())
      if let record = try await repository.fetchDayRecord(date: today) {
        ctx.currentDayRecord = record
      } else {
        ctx.currentDayRecord = try await repository.createDayRecord(date: today)
      }

      return reconcileInitialState(context: ctx)

    } catch {
      print("[ERROR] Initialization error: \(error)")
      return StateResult(nextState: self, updatedContext: ctx)
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    return StateResult(nextState: self, updatedContext: context)
  }

  @MainActor
  private func reconcileInitialState(context: WidgetContext) -> StateResult {
    var ctx = context
    let now = Date()

    guard let record = ctx.currentDayRecord else {
      ctx.isConfirmed = true
      print("[INIT] Complete: no record available")
      let nextState = ActiveState()
      return StateResult(nextState: nextState, updatedContext: ctx)
    }

    let currentPlanned = TimeLogic.getCurrentPlannedBlock(at: now, from: record.snapshotBlocks)
    let planned =
      currentPlanned ?? TimeLogic.getNextPlannedBlock(at: now, from: record.snapshotBlocks)
    ctx.plannedCategory = ctx.categories.first { $0.id == planned?.categoryId }

    if let actual = TimeLogic.getCurrentActualBlock(at: now, from: record.actualBlocks),
      let actualId = actual.categoryId
    {
      ctx.currentCategory = ctx.categories.first { $0.id == actualId }
    } else {
      ctx.currentCategory = ctx.plannedCategory
    }

    ctx.isConfirmed = true
    ctx.lastEventTime = Date()

    let isOnSchedule = ctx.currentCategory?.id == ctx.plannedCategory?.id
    print(
      "[INIT] System Status: \(isOnSchedule ? "ON-SCHEDULE" : "OFF-SCHEDULE"). Transitioning to \(isOnSchedule ? "Active" : "OffSchedule")State."
    )

    if !isOnSchedule {
      ctx.offsetExpiry = Date().addingTimeInterval(120)
    }

    let nextState: WidgetStateLogic = isOnSchedule ? ActiveState() : OffScheduleState()
    return StateResult(nextState: nextState, updatedContext: ctx)
  }
}

// MARK: - Concrete State: Active

final class ActiveState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .active

  func handle(
    action: WidgetAction,
    context: WidgetContext,
    repository: TodoPlannerRepository
  ) async -> StateResult {
    var ctx = context

    switch action {
    case .confirm:
      return await ctx.confirm(repo: repository)

    case .selectCategory(let category):
      let isPlanned = category.id == ctx.plannedCategory?.id
      return await ctx.transitionTo(category: category, isPlanned: isPlanned, repo: repository)

    case .syncToPlan:
      guard let planned = ctx.plannedCategory else {
        print("[ACTION] syncToPlan ignored: no planned category")
        return StateResult(nextState: self, updatedContext: ctx)
      }
      return await ctx.transitionTo(category: planned, isPlanned: true, repo: repository)

    case .spacePressed:
      if ctx.currentCategory?.hasPomodoroEnabled == true, ctx.isConfirmed {
        ctx.togglePomodoro()
      }
      return StateResult(nextState: self, updatedContext: ctx)

    case .initialize, .adjustOffset:
      return StateResult(nextState: self, updatedContext: ctx)
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    var ctx = context

    // Boundary Detection
    if let newBlock = currentPlannedBlock,
      newBlock.id != ctx.lastCheckedBlockId,
      TimeLogic.isWithinConfirmationWindow(for: newBlock, at: Date()),
      ctx.isConfirmed
    {
      print("[BOUNDARY] New block detected: \(newBlock.id). Prompting confirmation.")
      ctx.markBlockAsChecked(newBlock.id)
      ctx.isConfirmed = false
      NotificationCenter.default.post(name: .confirmationNeeded, object: nil)
      let nextState = ConfirmationPromptState()
      return StateResult(nextState: nextState, updatedContext: ctx)
    }

    // Pomodoro Logic
    if ctx.currentCategory?.hasPomodoroEnabled == true, ctx.isConfirmed {
      ctx.tickPomodoro()
    }

    return StateResult(nextState: self, updatedContext: ctx)
  }
}

// MARK: - Concrete State: Confirmation Prompt

final class ConfirmationPromptState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .confirmationPrompt

  func handle(
    action: WidgetAction,
    context: WidgetContext,
    repository: TodoPlannerRepository
  ) async -> StateResult {
    var ctx = context

    switch action {
    case .confirm, .spacePressed:
      return await ctx.confirm(repo: repository)

    case .selectCategory(let category):
      let isPlanned = category.id == ctx.plannedCategory?.id
      return await ctx.transitionTo(category: category, isPlanned: isPlanned, repo: repository)

    case .syncToPlan:
      guard let planned = ctx.plannedCategory else {
        return StateResult(nextState: self, updatedContext: ctx)
      }
      return await ctx.transitionTo(category: planned, isPlanned: true, repo: repository)

    default:
      return StateResult(nextState: self, updatedContext: ctx)
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    return StateResult(nextState: self, updatedContext: context)
  }
}

// MARK: - Concrete State: Off Schedule

final class OffScheduleState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .offSchedule

  func handle(
    action: WidgetAction,
    context: WidgetContext,
    repository: TodoPlannerRepository
  ) async -> StateResult {
    var ctx = context

    switch action {
    case .syncToPlan:
      guard let planned = ctx.plannedCategory else {
        return StateResult(nextState: self, updatedContext: ctx)
      }
      return await ctx.transitionTo(category: planned, isPlanned: true, repo: repository)

    case .adjustOffset(let minutes):
      let retroactiveTime = ctx.lastEventTime.addingTimeInterval(TimeInterval(-minutes * 60))
      ctx.offsetMinutes += minutes
      ctx.lastEventTime = retroactiveTime
      ctx.offsetExpiry = Date().addingTimeInterval(120)

      let blocks = await logEvent(
        type: .transition, category: ctx.currentCategory, occurredAt: retroactiveTime,
        recordId: ctx.currentDayRecord?.id, repo: repository)
      ctx.updateBlocks(blocks)

      print("[OFFSET] Applied \(minutes)m, total=\(ctx.offsetMinutes)m")
      return StateResult(nextState: self, updatedContext: ctx)

    case .selectCategory(let category):
      let isPlanned = category.id == ctx.plannedCategory?.id
      return await ctx.transitionTo(category: category, isPlanned: isPlanned, repo: repository)

    case .confirm:
      return await ctx.confirm(repo: repository)

    default:
      return StateResult(nextState: self, updatedContext: ctx)
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    var ctx = context
    let now = Date()

    // Offset Window Expiry
    if let expiry = ctx.offsetExpiry, now > expiry {
      print("[OFF-SCHEDULE] Offset window expired")
      ctx.offsetExpiry = nil
    }

    // Boundary Detection
    if let newBlock = currentPlannedBlock,
      newBlock.id != ctx.lastCheckedBlockId,
      TimeLogic.isWithinConfirmationWindow(for: newBlock, at: now)
    {
      print("[BOUNDARY] New block detected while off-schedule: \(newBlock.id)")
      ctx.markBlockAsChecked(newBlock.id)
      ctx.isConfirmed = false
      NotificationCenter.default.post(name: .confirmationNeeded, object: nil)
      let nextState = ConfirmationPromptState()
      return StateResult(nextState: nextState, updatedContext: ctx)
    }

    return StateResult(nextState: self, updatedContext: ctx)
  }
}

// MARK: - Modern Store Implementation

@Observable
@MainActor
final class WidgetStateStore {
  // --- Source of Truth ---
  private var currentState: WidgetStateLogic = InitializingState()
  private var context = WidgetContext()
  private let repository: TodoPlannerRepository

  // --- UI Projections (Glanceable Data) ---
  var displayState: WidgetStateIdentity = .initializing
  var categories: [Category] = []
  var currentDayRecord: DayRecord?
  var currentCategory: Category?
  var plannedCategory: Category?
  var lastEventTime = Date()
  var progressPercentage = 0.0
  var currentDuration = "0:00"
  var isConfirmed = false
  var offsetMinutes = 0
  var plannedDurationMinutes = 0
  var showOffsetBar = false
  var pomodoroState: PomodoroState?
  var pomodoroProgress = 0.0

  // --- Internal State ---
  private var ticker: AnyCancellable?
  private var currentPlannedBlock: PlannedBlock?

  var pomodoroActive: Bool {
    displayState == .active && isConfirmed
      && context.currentCategory?.hasPomodoroEnabled == true
  }

  var pomodoroPulsing: Bool { pomodoroActive && pomodoroProgress >= 1.0 }

  init(repository: TodoPlannerRepository) {
    self.repository = repository
    setupTicker()
  }

  convenience init() {
    self.init(repository: RepositoryFactory.createRepository())
  }

  // MARK: - Intent Dispatcher

  func dispatch(_ action: WidgetAction) async {
    print(
      "[ACTION] Received: \(actionDescription(action)); state=\(stateDescription(displayState))")

    let result = await currentState.handle(
      action: action,
      context: context,
      repository: repository
    )
    apply(result)

    print(
      "[ACTION] Completed: \(actionDescription(action)); state=\(stateDescription(displayState))")
  }

  // MARK: - Compatibility API

  func initialize() async { await dispatch(.initialize) }
  func confirmPlanned() async { await dispatch(.confirm) }
  func transitionToCategory(_ category: Category) async {
    await dispatch(.selectCategory(category))
  }
  func syncToPlan() async { await dispatch(.syncToPlan) }
  func adjustOffset(minutes: Int) async { await dispatch(.adjustOffset(minutes)) }
  func handleSpaceKey() async { await dispatch(.spacePressed) }

  // MARK: - State Application & Projection

  private func apply(_ result: StateResult) {
    self.context = result.updatedContext

    if result.nextState.identity != self.displayState {
      self.currentState = result.nextState
      self.displayState = result.nextState.identity
      updateMenuBarIcon()
    }

    updateCurrentState()
    projectContextToUI()
  }

  private func projectContextToUI() {
    // Direct mappings
    categories = context.categories
    currentDayRecord = context.currentDayRecord
    currentCategory = context.currentCategory
    plannedCategory = context.plannedCategory
    lastEventTime = context.lastEventTime
    isConfirmed = context.isConfirmed
    offsetMinutes = context.offsetMinutes

    let now = Date()

    // Duration string
    let elapsed = max(0, Int(now.timeIntervalSince(context.lastEventTime)))
    currentDuration = String(format: "%d:%02d", elapsed / 60, elapsed % 60)

    // Progress percentage
    if let planned = currentPlannedBlock {
      progressPercentage = TimeLogic.calculateProgress(for: planned, at: now)
    } else {
      progressPercentage = 0.0
    }

    // Offset bar visibility
    showOffsetBar =
      context.offsetExpiry.map { now < $0 } ?? false

    // Pomodoro projection
    if let config = context.currentCategory?.pomodoroConfig {
      let limit = context.pomodoroPhase == .work ? config.workDuration : config.restDuration
      pomodoroProgress =
        limit > 0 ? min(Double(context.pomodoroElapsed) / Double(limit), 1.0) : 0.0
      pomodoroState = PomodoroState(
        phase: context.pomodoroPhase, elapsed: context.pomodoroElapsed)
    } else {
      pomodoroState = nil
      pomodoroProgress = 0.0
    }
  }

  private func updateCurrentState() {
    guard let record = context.currentDayRecord else { return }
    let now = Date()

    let currentPlanned = TimeLogic.getCurrentPlannedBlock(at: now, from: record.snapshotBlocks)
    let planned =
      currentPlanned ?? TimeLogic.getNextPlannedBlock(at: now, from: record.snapshotBlocks)
    currentPlannedBlock = planned

    if let planned {
      plannedDurationMinutes = planned.durationMinutes
      context.plannedCategory = context.categories.first { $0.id == planned.categoryId }
    }
  }

  // MARK: - Heartbeat

  private func setupTicker() {
    ticker = Timer.publish(every: 1.0, on: .main, in: .common)
      .autoconnect()
      .sink { [weak self] _ in
        guard let self = self else { return }
        let result = self.currentState.onTick(
          context: self.context, currentPlannedBlock: self.currentPlannedBlock)
        self.apply(result)
      }
  }

  func startPeriodicRefresh() {
    if ticker == nil { setupTicker() }
  }

  func stopPeriodicRefresh() {
    ticker?.cancel()
    ticker = nil
  }

  // MARK: - System Integration

  private func updateMenuBarIcon() {
    let icon: MenuBarManager.IconState =
      switch displayState {
      case .initializing: .active
      case .confirmationPrompt: .confirmationNeeded
      case .active: .active
      case .offSchedule: .offSchedule
      }
    MenuBarManager.shared.updateIcon(state: icon, category: context.currentCategory)
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
}

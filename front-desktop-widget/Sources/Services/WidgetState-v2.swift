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
  var pomodoroPhase: PomodoroPhase = .work
  var pomodoroElapsed = 0
  var offsetMinutes = 0
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

enum WidgetEffect {
  case logTransition(category: Category, occurredAt: Date?)
  case logConfirmation(category: Category)
  case postNotification(Notification.Name)
  case updateMenuBarIcon
}

// MARK: - State Protocol & Result

/// The atomic output of a state operation.
struct StateResult {
  let nextState: WidgetStateLogic
  let updatedContext: WidgetContext
  let effects: [WidgetEffect]
}

/// The blueprint for all widget state logic modules.
@MainActor
protocol WidgetStateLogic {
  var identity: WidgetStateIdentity { get }

  /// Processes user intents (e.g., button clicks, key presses).
  func handle(action: WidgetAction, context: WidgetContext) -> StateResult

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

// MARK: - Pure State Helpers

func tickPomodoro(_ context: inout WidgetContext) {
  guard let config = context.currentCategory?.pomodoroConfig else { return }

  context.pomodoroElapsed += 1
  let limit = context.pomodoroPhase == .work ? config.workDuration : config.restDuration
  if context.pomodoroPhase == .rest && context.pomodoroElapsed > Int(Double(limit) * 1.5) {
    context.pomodoroPhase = .work
    context.pomodoroElapsed = 0
  }
}

func togglePomodoro(_ context: inout WidgetContext) {
  guard let config = context.currentCategory?.pomodoroConfig else { return }

  if context.pomodoroPhase == .work && context.pomodoroElapsed >= config.workDuration {
    context.pomodoroPhase = .rest
    context.pomodoroElapsed = 0
  } else if context.pomodoroPhase == .rest {
    context.pomodoroPhase = .work
    context.pomodoroElapsed = 0
  }
}

@MainActor
func confirmationBoundaryResult(
  context: WidgetContext,
  currentPlannedBlock: PlannedBlock?
) -> StateResult? {
  guard let newBlock = currentPlannedBlock,
    newBlock.id != context.lastCheckedBlockId,
    TimeLogic.isWithinConfirmationWindow(for: newBlock, at: Date())
  else { return nil }

  var updatedContext = context
  updatedContext.lastCheckedBlockId = newBlock.id
  return StateResult(
    nextState: ConfirmationPromptState(),
    updatedContext: updatedContext,
    effects: [.updateMenuBarIcon, .postNotification(.confirmationNeeded)]
  )
}

@MainActor
func transitionResult(context: WidgetContext, category: Category) -> StateResult {
  var updatedContext = context
  updatedContext.currentCategory = category
  updatedContext.lastEventTime = Date()
  updatedContext.pomodoroPhase = .work
  updatedContext.pomodoroElapsed = 0
  updatedContext.offsetMinutes = 0

  let isPlanned = category.id == context.plannedCategory?.id
  let nextState: WidgetStateLogic = isPlanned ? ActiveState() : OffScheduleState()
  return StateResult(
    nextState: nextState,
    updatedContext: updatedContext,
    effects: [.updateMenuBarIcon, .logTransition(category: category, occurredAt: nil)]
  )
}

@MainActor
func confirmationResult(context: WidgetContext, nextState: WidgetStateLogic) -> StateResult {
  var updatedContext = context
  updatedContext.lastEventTime = Date()
  guard let plannedCategory = updatedContext.plannedCategory else {
    return StateResult(nextState: nextState, updatedContext: updatedContext, effects: [])
  }
  return StateResult(
    nextState: nextState,
    updatedContext: updatedContext,
    effects: [.updateMenuBarIcon, .logConfirmation(category: plannedCategory)]
  )
}

func updateDayRecord(_ context: inout WidgetContext, with blocks: [ActualBlock]) {
  guard let record = context.currentDayRecord else { return }
  context.currentDayRecord = DayRecord(
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

  func handle(action: WidgetAction, context: WidgetContext) -> StateResult {
    guard case .initialize = action else {
      return StateResult(nextState: self, updatedContext: context, effects: [])
    }

    return reconcileInitialState(context: context)
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    return StateResult(nextState: self, updatedContext: context, effects: [])
  }

  private func reconcileInitialState(context: WidgetContext) -> StateResult {
    var ctx = context
    let now = Date()

    guard let record = ctx.currentDayRecord else {
      print("[INIT] Complete: no record available")
      let nextState = ActiveState()
      return StateResult(nextState: nextState, updatedContext: ctx, effects: [])
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

    ctx.lastEventTime = Date()

    let isOnSchedule = ctx.currentCategory?.id == ctx.plannedCategory?.id
    print(
      "[INIT] System Status: \(isOnSchedule ? "ON-SCHEDULE" : "OFF-SCHEDULE"). Transitioning to \(isOnSchedule ? "Active" : "OffSchedule")State."
    )

    let nextState: WidgetStateLogic = isOnSchedule ? ActiveState() : OffScheduleState()
    return StateResult(nextState: nextState, updatedContext: ctx, effects: [])
  }
}

// MARK: - Concrete State: Active

final class ActiveState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .active

  func handle(action: WidgetAction, context: WidgetContext) -> StateResult {
    var ctx = context

    switch action {
    case .confirm:
      return confirmationResult(context: ctx, nextState: ActiveState())

    case .selectCategory(let category):
      return transitionResult(context: ctx, category: category)

    case .syncToPlan:
      guard let planned = ctx.plannedCategory else {
        print("[ACTION] syncToPlan ignored: no planned category")
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      return transitionResult(context: ctx, category: planned)

    case .spacePressed:
      if ctx.currentCategory?.hasPomodoroEnabled == true {
        togglePomodoro(&ctx)
      }
      return StateResult(nextState: self, updatedContext: ctx, effects: [])

    case .initialize, .adjustOffset:
      return StateResult(nextState: self, updatedContext: ctx, effects: [])
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    var ctx = context

    if let boundary = confirmationBoundaryResult(
      context: ctx, currentPlannedBlock: currentPlannedBlock)
    {
      return boundary
    }

    // Pomodoro Logic
    if ctx.currentCategory?.hasPomodoroEnabled == true {
      tickPomodoro(&ctx)
    }

    return StateResult(nextState: self, updatedContext: ctx, effects: [])
  }
}

// MARK: - Concrete State: Confirmation Prompt

final class ConfirmationPromptState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .confirmationPrompt

  func handle(action: WidgetAction, context: WidgetContext) -> StateResult {
    let ctx = context

    switch action {
    case .confirm, .spacePressed:
      return confirmationResult(context: ctx, nextState: ActiveState())

    case .selectCategory(let category):
      return transitionResult(context: ctx, category: category)

    case .syncToPlan:
      guard let planned = ctx.plannedCategory else {
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      return transitionResult(context: ctx, category: planned)

    default:
      return StateResult(nextState: self, updatedContext: ctx, effects: [])
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    return StateResult(nextState: self, updatedContext: context, effects: [])
  }
}

// MARK: - Concrete State: Off Schedule

final class OffScheduleState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .offSchedule

  func handle(action: WidgetAction, context: WidgetContext) -> StateResult {
    var ctx = context

    switch action {
    case .syncToPlan:
      guard let planned = ctx.plannedCategory else {
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      return transitionResult(context: ctx, category: planned)

    case .adjustOffset(let minutes):
      let retroactiveTime = ctx.lastEventTime.addingTimeInterval(TimeInterval(-minutes * 60))
      ctx.offsetMinutes += minutes
      ctx.lastEventTime = retroactiveTime

      print("[OFFSET] Applied \(minutes)m, total=\(ctx.offsetMinutes)m")
      guard let category = ctx.currentCategory else {
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      return StateResult(
        nextState: self,
        updatedContext: ctx,
        effects: [.logTransition(category: category, occurredAt: retroactiveTime)]
      )

    case .selectCategory(let category):
      return transitionResult(context: ctx, category: category)

    case .confirm:
      return confirmationResult(context: ctx, nextState: ActiveState())

    default:
      return StateResult(nextState: self, updatedContext: ctx, effects: [])
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    return confirmationBoundaryResult(context: context, currentPlannedBlock: currentPlannedBlock)
      ?? StateResult(nextState: self, updatedContext: context, effects: [])
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
  private var tick = 0

  // --- UI Projections (Glanceable Data) ---
  var displayState: WidgetStateIdentity { currentState.identity }
  var categories: [Category] { context.categories }
  var currentDayRecord: DayRecord? { context.currentDayRecord }
  var currentCategory: Category? { context.currentCategory }
  var plannedCategory: Category? {
    guard let record = context.currentDayRecord else { return nil }
    let block =
      TimeLogic.getCurrentPlannedBlock(at: Date(), from: record.snapshotBlocks)
      ?? TimeLogic.getNextPlannedBlock(at: Date(), from: record.snapshotBlocks)
    return context.categories.first { $0.id == block?.categoryId }
  }
  var lastEventTime: Date { context.lastEventTime }
  var offsetMinutes: Int { context.offsetMinutes }
  var currentPlannedBlock: PlannedBlock? {
    _ = tick
    guard let record = context.currentDayRecord else { return nil }
    let now = Date()
    return TimeLogic.getCurrentPlannedBlock(at: now, from: record.snapshotBlocks)
      ?? TimeLogic.getNextPlannedBlock(at: now, from: record.snapshotBlocks)
  }
  var plannedDurationMinutes: Int { currentPlannedBlock?.durationMinutes ?? 0 }
  var progressPercentage: Double {
    _ = tick
    guard let planned = currentPlannedBlock else { return 0.0 }
    return TimeLogic.calculateProgress(for: planned, at: Date())
  }
  var currentDuration: String {
    _ = tick
    let elapsed = max(0, Int(Date().timeIntervalSince(lastEventTime)))
    return String(format: "%d:%02d", elapsed / 60, elapsed % 60)
  }
  var pomodoroState: PomodoroState? {
    guard context.currentCategory?.pomodoroConfig != nil else { return nil }
    return PomodoroState(phase: context.pomodoroPhase, elapsed: context.pomodoroElapsed)
  }
  var pomodoroProgress: Double {
    let config = context.currentCategory?.pomodoroConfig
    let limit = context.pomodoroPhase == .work ? config?.workDuration : config?.restDuration
    guard let limit, limit > 0 else { return 0.0 }
    return min(Double(context.pomodoroElapsed) / Double(limit), 1.0)
  }

  // --- Internal State ---
  private var ticker: AnyCancellable?

  var pomodoroActive: Bool {
    displayState == .active && context.currentCategory?.hasPomodoroEnabled == true
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

    context.plannedCategory = plannedCategory
    let result = currentState.handle(action: action, context: context)
    await apply(result)

    print(
      "[ACTION] Completed: \(actionDescription(action)); state=\(stateDescription(displayState))")
  }

  // MARK: - Compatibility API

  func initialize() async {
    do {
      context.categories = try await repository.fetchCategories()
      let today = DateFormatter.yyyyMMdd.string(from: Date())
      if let record = try await repository.fetchDayRecord(date: today) {
        context.currentDayRecord = record
      } else {
        context.currentDayRecord = try await repository.createDayRecord(date: today)
      }
      await dispatch(.initialize)
    } catch {
      print("[ERROR] Initialization error: \(error)")
    }
  }
  func confirmPlanned() async { await dispatch(.confirm) }
  func transitionToCategory(_ category: Category) async {
    await dispatch(.selectCategory(category))
  }
  func syncToPlan() async { await dispatch(.syncToPlan) }
  func adjustOffset(minutes: Int) async { await dispatch(.adjustOffset(minutes)) }
  func handleSpaceKey() async { await dispatch(.spacePressed) }

  // MARK: - State Application & Projection

  private func apply(_ result: StateResult) async {
    self.context = result.updatedContext

    self.currentState = result.nextState

    for effect in result.effects {
      await execute(effect)
    }
  }

  private func execute(_ effect: WidgetEffect) async {
    switch effect {
    case .logTransition(let category, let occurredAt):
      let blocks = await logEvent(
        type: .transition,
        category: category,
        occurredAt: occurredAt,
        recordId: context.currentDayRecord?.id,
        repo: repository
      )
      updateDayRecord(&context, with: blocks)

    case .logConfirmation(let category):
      let blocks = await logEvent(
        type: .confirmation,
        category: category,
        occurredAt: nil,
        recordId: context.currentDayRecord?.id,
        repo: repository
      )
      updateDayRecord(&context, with: blocks)

    case .postNotification(let name):
      NotificationCenter.default.post(name: name, object: nil)

    case .updateMenuBarIcon:
      updateMenuBarIcon()
    }
  }

  // MARK: - Heartbeat

  private func setupTicker() {
    ticker = Timer.publish(every: 1.0, on: .main, in: .common)
      .autoconnect()
      .sink { [weak self] _ in
        guard let self = self else { return }
        self.tick += 1
        self.context.plannedCategory = self.plannedCategory
        let result = self.currentState.onTick(
          context: self.context, currentPlannedBlock: self.currentPlannedBlock)
        Task { await self.apply(result) }
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

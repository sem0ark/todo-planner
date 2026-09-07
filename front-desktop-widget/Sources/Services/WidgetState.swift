import Combine
import Foundation
import Observation
import SwiftUI

// MARK: - Core Type Definitions

extension Notification.Name {
  static let confirmationNeeded = Notification.Name("confirmationNeeded")
  static let pomodoroCompleted = Notification.Name("pomodoroCompleted")
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
  case active
  case prompted
}

struct ScheduleDeviation {
  let expected: Category
  let actual: Category
  let deviatedAt: Date
}

struct WidgetContext {
  var categories: [Category] = []
  var currentDayRecord: DayRecord?
  var currentPlannedBlocks: [PlannedBlock] = []
  var currentCategory: Category?
  var plannedCategory: Category?
  var lastEventTime = Date()
  var lastEventClientId: String?
  var pomodoroPhase: PomodoroPhase = .work
  var pomodoroElapsed = 0
  var offsetMinutes = 0
  var lastCheckedBlockId: String?
}

enum WidgetAction {
  case initialize
  case reload
  case selectCategory(Category)
  case adjustOffset(Int)
  case primaryAction
  case returnToPlan
}

enum DayEventType: String {
  case confirmation
  case transition
  case amendment
}

enum WidgetEffect {
  case logTransition(category: Category, occurredAt: Date?)
  case logConfirmation(category: Category)
  case logAmendment(targetClientEventId: String, correctedAt: Date)
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
    guard let current = secondsSinceStartOfDay(for: time) else { return nil }
    return blocks.first { block in
      guard let begin = parseSeconds(from: block.startTime) else { return false }
      return current >= begin && current < begin + block.durationMinutes * 60
    }
  }

  static func getNextPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    guard let current = secondsSinceStartOfDay(for: time) else { return nil }
    return blocks.compactMap { block -> (PlannedBlock, Int)? in
      guard let start = parseSeconds(from: block.startTime), start > current else { return nil }
      return (block, start)
    }.min { $0.1 < $1.1 }?.0
  }

  static func getCurrentActualBlock(at time: Date, from blocks: [ActualBlock]) -> ActualBlock? {
    guard let current = secondsSinceStartOfDay(for: time) else { return nil }
    return blocks.last { block in
      guard let begin = parseSeconds(from: block.startTime) else { return false }
      let isOpenEnded = block.durationMinutes <= 0
      return current >= begin && (isOpenEnded || current < begin + block.durationMinutes * 60)
    }
  }

  static func calculateProgress(for block: PlannedBlock, at time: Date) -> Double {
    guard let current = secondsSinceStartOfDay(for: time),
      let begin = parseSeconds(from: block.startTime)
    else { return 0 }
    let elapsed = max(0, current - begin)
    return min(1, Double(elapsed) / Double(max(1, block.durationMinutes * 60)))
  }

  static func isWithinConfirmationWindow(for block: PlannedBlock, at time: Date) -> Bool {
    guard let current = secondsSinceStartOfDay(for: time),
      let begin = parseSeconds(from: block.startTime)
    else { return false }
    let elapsed = current - begin
    return elapsed >= 0 && elapsed < 60
  }

  static func secondsSinceStartOfDay(for date: Date) -> Int? {
    let components = Calendar.current.dateComponents([.hour, .minute, .second], from: date)
    guard let hour = components.hour, let minute = components.minute, let second = components.second
    else {
      WidgetLogger.error("Calendar date is missing time components")
      return nil
    }
    return hour * 3600 + minute * 60 + second
  }

  static func parseSeconds(from timeString: String) -> Int? {
    guard let seconds = TimeFormats.secondsSinceDayStart(timeString) else {
      WidgetLogger.error("Unable to parse schedule time", context: ["value": timeString])
      return nil
    }
    return seconds
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

enum EventLoggingError: Error {
  case missingCalendarDate
  case missingCategory(eventType: DayEventType)
  case incompleteAmendment
}

func tickPomodoro(_ context: inout WidgetContext) -> Bool {
  guard let config = context.currentCategory?.pomodoroConfig else { return false }

  context.pomodoroElapsed += 1
  let limit = (context.pomodoroPhase == .work ? config.workDuration : config.restDuration) * 60
  guard limit > 0 else { return false }

  if context.pomodoroPhase == .rest && context.pomodoroElapsed > Int(Double(limit) * 1.5) {
    context.pomodoroPhase = .work
    context.pomodoroElapsed = 0
  }

  return context.pomodoroElapsed < limit && context.pomodoroElapsed + 1 >= limit
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
    nextState: PromptedState(),
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

  return StateResult(
    nextState: ActiveState(),
    updatedContext: updatedContext,
    effects: [.updateMenuBarIcon, .logTransition(category: category, occurredAt: nil)]
  )
}

@MainActor
func confirmationResult(context: WidgetContext, nextState: WidgetStateLogic) -> StateResult {
  var updatedContext = context
  guard let plannedCategory = updatedContext.plannedCategory else {
    WidgetLogger.error("Cannot confirm without a planned category")
    return StateResult(nextState: nextState, updatedContext: context, effects: [])
  }
  updatedContext.lastEventTime = Date()
  updatedContext.currentCategory = plannedCategory
  updatedContext.pomodoroPhase = .work
  updatedContext.pomodoroElapsed = 0
  return StateResult(
    nextState: nextState,
    updatedContext: updatedContext,
    effects: [.updateMenuBarIcon, .logConfirmation(category: plannedCategory)]
  )
}

func updateDayRecord(_ context: inout WidgetContext, with blocks: [ActualBlock]) {
  guard let record = context.currentDayRecord else { return }
  context.currentDayRecord = DayRecord(
    calendarDate: record.calendarDate,
    plan: record.plan,
    actual: blocks,
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
  calendarDate: String?,
  repo: TodoPlannerRepository,
  targetClientEventId: String? = nil,
  correctedAt: Date? = nil
) async throws -> (blocks: [ActualBlock], clientEventId: String) {
  guard let calendarDate else {
    throw EventLoggingError.missingCalendarDate
  }
  let categoryId = category?.id
  guard type == .amendment || categoryId != nil else {
    throw EventLoggingError.missingCategory(eventType: type)
  }
  guard type != .amendment || targetClientEventId != nil,
    type != .amendment || correctedAt != nil
  else {
    throw EventLoggingError.incompleteAmendment
  }

  let event = DayEvent(
    eventType: type.rawValue,
    categoryId: type == .amendment ? nil : categoryId,
    occurredAt: occurredAt ?? Date(),
    targetClientEventId: targetClientEventId,
    correctedAt: correctedAt
  )

  let response = try await repo.submitEvents(calendarDate: calendarDate, events: [event])
  print("[SYNC] Event persisted: \(type.rawValue), blocks=\(response.actual.count)")
  return (response.actual, event.clientEventId)
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

    let currentPlanned = TimeLogic.getCurrentPlannedBlock(at: now, from: ctx.currentPlannedBlocks)
    let planned =
      currentPlanned ?? TimeLogic.getNextPlannedBlock(at: now, from: ctx.currentPlannedBlocks)
    let plannedCategory = ctx.categories.first { $0.id == planned?.categoryId }
    ctx.plannedCategory = plannedCategory
    ctx.currentCategory = plannedCategory

    ctx.lastEventTime = Date()

    print("[INIT] System Status: Picked planned category on startup. Transitioning to ActiveState.")

    var effects: [WidgetEffect] = [.updateMenuBarIcon]
    if let category = plannedCategory {
      effects.append(.logTransition(category: category, occurredAt: nil))
    }

    return StateResult(nextState: ActiveState(), updatedContext: ctx, effects: effects)
  }
}

// MARK: - Concrete State: Active

final class ActiveState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .active

  func handle(action: WidgetAction, context: WidgetContext) -> StateResult {
    var ctx = context

    switch action {
    case .reload:
      return StateResult(nextState: InitializingState(), updatedContext: ctx, effects: [])

    case .primaryAction:
      if ctx.currentCategory?.hasPomodoroEnabled == true {
        togglePomodoro(&ctx)
      }
      return StateResult(nextState: self, updatedContext: ctx, effects: [])

    case .selectCategory(let category):
      return transitionResult(context: ctx, category: category)

    case .adjustOffset(let minutes):
      guard ctx.currentCategory != nil, let lastEventClientId = ctx.lastEventClientId else {
        WidgetLogger.error(
          "Cannot adjust offset without a persisted event", context: ["minutes": String(minutes)])
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      let retroactiveTime = ctx.lastEventTime.addingTimeInterval(TimeInterval(-minutes * 60))
      ctx.offsetMinutes += minutes
      ctx.lastEventTime = retroactiveTime
      return StateResult(
        nextState: self,
        updatedContext: ctx,
        effects: [
          .logAmendment(targetClientEventId: lastEventClientId, correctedAt: retroactiveTime)
        ]
      )

    case .returnToPlan:
      guard let planned = ctx.plannedCategory, planned.id != ctx.currentCategory?.id else {
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      return transitionResult(context: ctx, category: planned)

    case .initialize:
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
      if tickPomodoro(&ctx) {
        var effects: [WidgetEffect] = [.postNotification(.pomodoroCompleted)]
        if let category = ctx.currentCategory {
          effects.append(.logConfirmation(category: category))
        }
        return StateResult(
          nextState: self,
          updatedContext: ctx,
          effects: effects
        )
      }
    }

    return StateResult(nextState: self, updatedContext: ctx, effects: [])
  }
}

// MARK: - Concrete State: Confirmation Prompt

final class PromptedState: WidgetStateLogic {
  let identity: WidgetStateIdentity = .prompted

  func handle(action: WidgetAction, context: WidgetContext) -> StateResult {
    let ctx = context

    switch action {
    case .reload:
      return StateResult(nextState: InitializingState(), updatedContext: ctx, effects: [])

    case .primaryAction:
      return confirmationResult(context: ctx, nextState: ActiveState())

    case .selectCategory(let category):
      return transitionResult(context: ctx, category: category)

    default:
      return StateResult(nextState: self, updatedContext: ctx, effects: [])
    }
  }

  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult {
    return StateResult(nextState: self, updatedContext: context, effects: [])
  }
}

// MARK: - Modern Store Implementation

@Observable
@MainActor
class WidgetStateStore {
  // --- Source of Truth ---
  var currentState: WidgetStateLogic = InitializingState()
  var context = WidgetContext()
  private let repository: TodoPlannerRepository
  private var tick = 0
  private var didSynchronizeAtStartup = false

  // --- UI Projections (Glanceable Data) ---
  var displayState: WidgetStateIdentity { currentState.identity }
  var categories: [Category] { context.categories }
  var currentDayRecord: DayRecord? { context.currentDayRecord }
  var currentCategory: Category? { context.currentCategory }
  var isOnSchedule: Bool {
    guard let current = context.currentCategory, let planned = plannedCategory else { return false }
    return current.id == planned.id
  }
  var scheduleDeviation: ScheduleDeviation? {
    guard !isOnSchedule, let planned = plannedCategory, let actual = context.currentCategory else {
      return nil
    }
    return ScheduleDeviation(expected: planned, actual: actual, deviatedAt: context.lastEventTime)
  }
  var plannedCategory: Category? {
    let block =
      TimeLogic.getCurrentPlannedBlock(at: Date(), from: context.currentPlannedBlocks)
      ?? TimeLogic.getNextPlannedBlock(at: Date(), from: context.currentPlannedBlocks)
    return context.categories.first { $0.id == block?.categoryId }
  }
  var lastEventTime: Date { context.lastEventTime }
  var offsetMinutes: Int { context.offsetMinutes }
  var currentPlannedBlock: PlannedBlock? {
    _ = tick
    let now = Date()
    return TimeLogic.getCurrentPlannedBlock(at: now, from: context.currentPlannedBlocks)
      ?? TimeLogic.getNextPlannedBlock(at: now, from: context.currentPlannedBlocks)
  }
  var plannedDurationMinutes: Int { currentPlannedBlock?.durationMinutes ?? 0 }
  var remainingPlannedMinutes: Int {
    guard let plannedBlock = currentPlannedBlock else { return 0 }
    guard let currentSeconds = TimeLogic.secondsSinceStartOfDay(for: Date()),
      let plannedStartSeconds = TimeLogic.parseSeconds(from: plannedBlock.startTime)
    else { return 0 }
    let elapsedSeconds = max(
      0,
      currentSeconds - plannedStartSeconds
    )
    let remainingSeconds = max(0, plannedBlock.durationMinutes * 60 - elapsedSeconds)
    return Int(ceil(Double(remainingSeconds) / 60.0))
  }
  var progressPercentage: Double {
    _ = tick
    guard let planned = currentPlannedBlock else { return 0.0 }
    return TimeLogic.calculateProgress(for: planned, at: Date())
  }
  var pomodoroState: PomodoroState? {
    guard context.currentCategory?.pomodoroConfig != nil else { return nil }
    return PomodoroState(phase: context.pomodoroPhase, elapsed: context.pomodoroElapsed)
  }
  var pomodoroProgress: Double {
    guard let config = context.currentCategory?.pomodoroConfig else { return 0.0 }
    let limit = (context.pomodoroPhase == .work ? config.workDuration : config.restDuration) * 60
    guard limit > 0 else { return 0.0 }
    return min(Double(context.pomodoroElapsed) / Double(limit), 1.0)
  }

  // --- Internal State ---
  private var ticker: AnyCancellable?
  var lastError: String?

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
      try await loadData()
      await dispatch(.initialize)
    } catch {
      lastError = String(describing: error)
      WidgetLogger.error(
        "Initialization failed; widget remains inactive", context: ["error": lastError!])
    }
  }

  func reload() async {
    await dispatch(.reload)

    do {
      try await loadData()
      await dispatch(.initialize)
    } catch {
      lastError = String(describing: error)
      WidgetLogger.error("Reload failed; widget remains inactive", context: ["error": lastError!])
    }
  }

  private func loadData() async throws {
    let today = DateFormatter.yyyyMMdd.string(from: Date())
    let bootstrap = try await repository.initialize(calendarDate: today)
    try validateBootstrap(bootstrap, requestedDate: today)

    context.categories = bootstrap.categories
    context.currentPlannedBlocks = bootstrap.dayRecord.plan
    context.currentDayRecord = bootstrap.dayRecord
  }

  private func validateBootstrap(_ bootstrap: InitResponse, requestedDate: String) throws {
    guard bootstrap.dayRecord.calendarDate == requestedDate else {
      throw StorageError.invalidContract("day_record calendar date does not match request")
    }
    let categoryIds = Set(bootstrap.categories.map(\.id))
    for block in bootstrap.dayRecord.plan {
      guard categoryIds.contains(block.categoryId), block.durationMinutes > 0,
        TimeLogic.parseSeconds(from: block.startTime) != nil
      else {
        throw StorageError.invalidContract("invalid planned block \(block.id)")
      }
    }
    for block in bootstrap.dayRecord.actual {
      let hasValidCategory =
        if let categoryId = block.categoryId {
          categoryIds.contains(categoryId)
        } else {
          block.blockType == "untracked"
        }
      guard hasValidCategory, TimeLogic.parseSeconds(from: block.startTime) != nil else {
        throw StorageError.invalidContract("invalid actual block \(block.id)")
      }
    }
  }

  func handleSelectCategory(_ category: Category) async {
    await dispatch(.selectCategory(category))
  }
  func handlePrimaryAction() async { await dispatch(.primaryAction) }
  func adjustOffset(minutes: Int) async { await dispatch(.adjustOffset(minutes)) }

  func synchronize() async {
    do {
      try await repository.synchronize()
      lastError = nil
    } catch {
      lastError = String(describing: error)
      WidgetLogger.error("Synchronization failed", context: ["error": lastError!])
    }
  }

  func synchronizeOnStartup() async {
    guard !didSynchronizeAtStartup else { return }
    await synchronize()
    if lastError == nil {
      didSynchronizeAtStartup = true
    }
  }

  // MARK: - State Application & Projection

  func apply(_ result: StateResult) async {
    let previousContext = context
    let previousState = currentState
    self.context = result.updatedContext
    self.currentState = result.nextState

    for effect in result.effects {
      guard await execute(effect) else {
        context = previousContext
        currentState = previousState
        return
      }
    }
  }

  private func execute(_ effect: WidgetEffect) async -> Bool {
    switch effect {
    case .logTransition(let category, let occurredAt):
      logEffectContext("logTransition", eventCategory: category)
      do {
        let eventResult = try await logEvent(
          type: .transition,
          category: category,
          occurredAt: occurredAt,
          calendarDate: context.currentDayRecord?.calendarDate,
          repo: repository
        )
        context.lastEventClientId = eventResult.clientEventId
        if !eventResult.blocks.isEmpty {
          updateDayRecord(&context, with: eventResult.blocks)
        }
      } catch {
        lastError = String(describing: error)
        WidgetLogger.error(
          "Failed to log transition; local state rolled back", context: ["error": lastError!])
        return false
      }

    case .logConfirmation(let category):
      logEffectContext("logConfirmation", eventCategory: category)
      do {
        let eventResult = try await logEvent(
          type: .confirmation,
          category: category,
          occurredAt: nil,
          calendarDate: context.currentDayRecord?.calendarDate,
          repo: repository
        )
        context.lastEventClientId = eventResult.clientEventId
        if !eventResult.blocks.isEmpty {
          updateDayRecord(&context, with: eventResult.blocks)
        }
      } catch {
        lastError = String(describing: error)
        WidgetLogger.error(
          "Failed to log confirmation; local state rolled back", context: ["error": lastError!])
        return false
      }

    case .logAmendment(let targetClientEventId, let correctedAt):
      do {
        let eventResult = try await logEvent(
          type: .amendment,
          category: nil,
          // Keep amendment ordering separate from the corrected timestamp. The
          // server uses occurred_at to select the latest amendment for a target.
          occurredAt: Date(),
          calendarDate: context.currentDayRecord?.calendarDate,
          repo: repository,
          targetClientEventId: targetClientEventId,
          correctedAt: correctedAt
        )
        if !eventResult.blocks.isEmpty {
          updateDayRecord(&context, with: eventResult.blocks)
        }
      } catch {
        lastError = String(describing: error)
        WidgetLogger.error(
          "Failed to amend event; local state rolled back", context: ["error": lastError!])
        return false
      }

    case .postNotification(let name):
      NotificationCenter.default.post(name: name, object: nil)

    case .updateMenuBarIcon:
      updateMenuBarIcon()
    }
    return true
  }

  private func logEffectContext(_ effectName: String, eventCategory: Category) {
    let plannedBlock = currentPlannedBlock
    let actualBlock = context.currentDayRecord.flatMap {
      TimeLogic.getCurrentActualBlock(at: Date(), from: $0.actual)
    }

    let message =
      "[EFFECT] \(effectName) eventCategory=\(categoryDescription(eventCategory)); "
      + "state=\(stateDescription(displayState)); currentCategory=\(categoryDescription(context.currentCategory)); "
      + "plannedCategory=\(categoryDescription(context.plannedCategory)); plannedBlock=\(plannedBlockDescription(plannedBlock)); "
      + "actualBlock=\(actualBlockDescription(actualBlock)); offsetMinutes=\(context.offsetMinutes)"
    print(message)
  }

  private func categoryDescription(_ category: Category?) -> String {
    guard let category else { return "nil" }
    return "id=\(category.id),name=\(category.name)"
  }

  private func plannedBlockDescription(_ block: PlannedBlock?) -> String {
    guard let block else { return "nil" }
    return
      "id=\(block.id),categoryId=\(block.categoryId),start=\(block.startTime),duration=\(block.durationMinutes)m"
  }

  private func actualBlockDescription(_ block: ActualBlock?) -> String {
    guard let block else { return "nil" }
    return
      "id=\(block.id),categoryId=\(block.categoryId.map(String.init) ?? "nil"),type=\(block.blockType),start=\(block.startTime),duration=\(block.durationMinutes)m"
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

  func updateMenuBarIcon() {
    let icon: MenuBarManager.IconState =
      switch displayState {
      case .initializing: .active
      case .prompted: .confirmationNeeded
      case .active: .active
      }
    let iconCategory =
      displayState == .prompted
      ? context.plannedCategory ?? context.currentCategory
      : context.currentCategory
    MenuBarManager.shared.updateIcon(state: icon, category: iconCategory)
  }

  private func stateDescription(_ state: WidgetStateIdentity) -> String {
    switch state {
    case .initializing: "initializing"
    case .prompted: "prompted"
    case .active: "active"
    }
  }

  private func actionDescription(_ action: WidgetAction) -> String {
    switch action {
    case .initialize: "initialize"
    case .reload: "reload"
    case .selectCategory(let category): "selectCategory(\(category.name))"
    case .adjustOffset(let minutes): "adjustOffset(\(minutes)m)"
    case .primaryAction: "primaryAction"
    case .returnToPlan: "returnToPlan"
    }
  }
}

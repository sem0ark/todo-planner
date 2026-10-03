import Combine
import Foundation
import Observation
import SwiftUI

private let widgetTickerIntervalSeconds = 5

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

enum WidgetScreenState: Equatable {
  case checkingAuthentication
  case login
  case widget
  case error(String)
}

struct ScheduleDeviation {
  let expected: Category
  let actual: Category
  let deviatedAt: Date
}

struct WidgetContext {
  var categories: [Category] = []
  var settings: UserSettings?
  var currentDayRecord: DayRecord?
  var currentPlannedBlocks: [PlannedBlock] = []
  var currentCategory: Category?
  var plannedCategory: Category?
  var lastEventTime = Date()
  var lastEventClientId: String?
  var eventLocalTimestamps: [String: String] = [:]
  var pomodoroPhase: PomodoroPhase = .work
  var pomodoroElapsed = 0
  var offsetSeconds = 0
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

  /// Processes temporal events (e.g., periodic heartbeat, boundary checks).
  func onTick(context: WidgetContext, currentPlannedBlock: PlannedBlock?) -> StateResult
}

// MARK: - Supporting Utilities

struct TimeLogic {
  static func getCurrentPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    guard let current = secondsSinceStartOfDay(for: time) else { return nil }
    return blocks.first { block in
      guard let begin = block.startSeconds else { return false }
      return current >= begin && current < begin + block.durationSeconds
    }
  }

  static func getNextPlannedBlock(at time: Date, from blocks: [PlannedBlock]) -> PlannedBlock? {
    guard let current = secondsSinceStartOfDay(for: time) else { return nil }
    return blocks.compactMap { block -> (PlannedBlock, Int)? in
      guard let start = block.startSeconds, start > current else { return nil }
      return (block, start)
    }.min { $0.1 < $1.1 }?.0
  }

  static func calculateProgress(for block: PlannedBlock, at time: Date) -> Double {
    guard let current = secondsSinceStartOfDay(for: time),
      let begin = block.startSeconds
    else { return 0 }
    let elapsed = max(0, current - begin)
    return min(1, Double(elapsed) / Double(max(1, block.durationSeconds)))
  }

  static func isWithinConfirmationWindow(for block: PlannedBlock, at time: Date) -> Bool {
    guard let current = secondsSinceStartOfDay(for: time),
      let begin = block.startSeconds
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

enum PomodoroTickOutcome {
  case none
  case workCompleted
  case restCompleted
}

func tickPomodoro(_ context: inout WidgetContext) -> PomodoroTickOutcome {
  guard let category = context.currentCategory, let config = category.pomodoroConfig
  else { return .none }

  let limit = context.pomodoroPhase == .work ? config.workDuration : config.restDuration
  guard limit > 0 else { return .none }

  let previousElapsed = context.pomodoroElapsed
  context.pomodoroElapsed += widgetTickerIntervalSeconds

  let autoSkipLimit = Int(Double(limit) * 1.5)
  let crossedWorkLimit =
    context.pomodoroPhase == .work && previousElapsed < limit && context.pomodoroElapsed >= limit

  WidgetLogger.debug(
    "Pomodoro tick",
    context: [
      "categoryId": String(category.id),
      "phase": String(describing: context.pomodoroPhase),
      "previousElapsedSeconds": String(previousElapsed),
      "elapsedSeconds": String(context.pomodoroElapsed),
      "limitSeconds": String(limit),
      "autoSkipSeconds": String(autoSkipLimit),
      "crossedWorkLimit": String(crossedWorkLimit),
    ])

  if context.pomodoroPhase == .rest && context.pomodoroElapsed >= autoSkipLimit {
    WidgetLogger.debug(
      "Pomodoro rest auto-skipped",
      context: [
        "categoryId": String(category.id), "elapsedSeconds": String(context.pomodoroElapsed),
        "autoSkipSeconds": String(autoSkipLimit),
      ])

    context.pomodoroPhase = .work
    context.pomodoroElapsed = 0
    return .restCompleted
  }

  return crossedWorkLimit ? .workCompleted : .none
}

func togglePomodoro(_ context: inout WidgetContext) {
  guard let category = context.currentCategory, let config = category.pomodoroConfig
  else { return }

  let workLimit = config.workDuration
  guard workLimit > 0 else { return }

  if context.pomodoroPhase == .work && context.pomodoroElapsed >= workLimit {
    context.pomodoroPhase = .rest
    context.pomodoroElapsed = 0
  } else if context.pomodoroPhase == .rest {
    context.pomodoroPhase = .work
    context.pomodoroElapsed = 0
  }

  WidgetLogger.debug(
    "Pomodoro toggled",
    context: [
      "categoryId": String(category.id),
      "phase": String(describing: context.pomodoroPhase),
      "elapsedSeconds": String(context.pomodoroElapsed),
      "workLimitSeconds": String(workLimit),
    ])
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
  updatedContext.offsetSeconds = 0

  return StateResult(
    nextState: ActiveState(),
    updatedContext: updatedContext,
    effects: [.updateMenuBarIcon, .logTransition(category: category, occurredAt: nil)]
  )
}

@MainActor
func confirmationResult(context: WidgetContext, nextState: WidgetStateLogic) -> StateResult {
  guard let plannedCategory = context.plannedCategory else {
    WidgetLogger.error("Cannot confirm without a planned category")
    return StateResult(nextState: nextState, updatedContext: context, effects: [])
  }
  return transitionResult(context: context, category: plannedCategory)
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
  correctedAt: Date? = nil,
  targetOccurredAtLocal: String? = nil
) async throws -> (clientEventId: String, event: DayEvent) {
  guard let calendarDate else {
    throw EventLoggingError.missingCalendarDate
  }
  let categoryId = category?.id
  guard type == .amendment || categoryId != nil else {
    throw EventLoggingError.missingCategory(eventType: type)
  }
  guard type != .amendment || targetClientEventId != nil,
    type != .amendment || correctedAt != nil,
    type != .amendment || targetOccurredAtLocal != nil
  else {
    throw EventLoggingError.incompleteAmendment
  }

  let eventOccurredAt = occurredAt ?? Date()
  let correctedAtLocal: String?
  if let correctedAt, let targetOccurredAtLocal {
    correctedAtLocal = TimeFormats.localTimestamp(
      for: correctedAt, preservingOffsetFrom: targetOccurredAtLocal)
  } else {
    correctedAtLocal = nil
  }
  let event = DayEvent(
    eventType: type.rawValue,
    categoryId: type == .amendment ? nil : categoryId,
    occurredAt: eventOccurredAt,
    targetClientEventId: targetClientEventId,
    correctedAt: correctedAt,
    correctedAtLocal: correctedAtLocal
  )

  _ = try await repo.submitEvents(calendarDate: calendarDate, events: [event])
  WidgetLogger.debug("Local event appended", context: ["eventType": type.rawValue])
  return (event.clientEventId, event)
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

    WidgetLogger.debug(
      "Picked planned category on startup; transitioning to active state",
      context: ["hasPlannedCategory": String(plannedCategory != nil)])

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
      var effects: [WidgetEffect] = []
      if let category = ctx.currentCategory {
        effects.append(.logConfirmation(category: category))
      }
      if ctx.currentCategory?.hasPomodoroEnabled == true {
        togglePomodoro(&ctx)
      }
      return StateResult(nextState: self, updatedContext: ctx, effects: effects)

    case .selectCategory(let category):
      return transitionResult(context: ctx, category: category)

    case .adjustOffset(let minutes):
      guard ctx.currentCategory != nil, let lastEventClientId = ctx.lastEventClientId else {
        WidgetLogger.error(
          "Cannot adjust offset without a persisted event", context: ["minutes": String(minutes)])
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      let proposedOffsetSeconds = ctx.offsetSeconds + minutes * 60
      guard proposedOffsetSeconds >= 0 else {
        WidgetLogger.debug(
          "Ignoring offset adjustment below zero",
          context: [
            "minutes": String(minutes),
            "currentOffsetSeconds": String(ctx.offsetSeconds),
          ])
        return StateResult(nextState: self, updatedContext: ctx, effects: [])
      }
      let retroactiveTime = ctx.lastEventTime.addingTimeInterval(TimeInterval(-minutes * 60))
      ctx.offsetSeconds += minutes * 60
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
      switch tickPomodoro(&ctx) {
      case .workCompleted:
        var effects: [WidgetEffect] = []
        if let category = ctx.currentCategory {
          effects.append(.logConfirmation(category: category))
        }
        effects.append(.postNotification(.pomodoroCompleted))
        return StateResult(
          nextState: self,
          updatedContext: ctx,
          effects: effects
        )

      case .restCompleted:
        let effects: [WidgetEffect] = [.postNotification(.pomodoroCompleted)]
        return StateResult(
          nextState: self,
          updatedContext: ctx,
          effects: effects
        )

      case .none:
        break
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
  var screenState: WidgetScreenState = .checkingAuthentication
  var context = WidgetContext()
  private let repository: TodoPlannerRepository
  private var tick = 0

  // --- UI Projections (Glanceable Data) ---
  var displayState: WidgetStateIdentity { currentState.identity }
  var categories: [Category] { context.categories }
  var currentDayRecord: DayRecord? { context.currentDayRecord }
  var settings: UserSettings? { context.settings }
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
  var offsetMinutes: Int { context.offsetSeconds / 60 }
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
    let limit = (context.pomodoroPhase == .work ? config.workDuration : config.restDuration)
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
    WidgetLogger.debug(
      "Action received",
      context: [
        "action": actionDescription(action),
        "state": stateDescription(displayState),
      ])

    context.plannedCategory = plannedCategory
    let result = currentState.handle(action: action, context: context)
    await apply(result)

    WidgetLogger.debug(
      "Action completed",
      context: [
        "action": actionDescription(action),
        "state": stateDescription(displayState),
      ])
  }

  // MARK: - Screen Lifecycle

  func start() async {
    guard screenState == .checkingAuthentication else { return }
    let hasAuthenticationToken =
      repository.getAuthToken().map {
        AuthController.isWorkingAuthenticationToken($0)
      } ?? false
    WidgetLogger.debug(
      "Starting widget",
      context: ["hasAuthenticationToken": String(hasAuthenticationToken)])

    if hasAuthenticationToken {
      await initialize()
      if screenState == .widget {
        startPeriodicRefresh()
      }
      return
    }

    if repository.getAuthToken() != nil {
      try? await repository.clearAuth()
    }

    if initializeFromCache() {
      startPeriodicRefresh()
    } else {
      screenState = .login
    }

    WidgetLogger.debug(
      "Widget startup completed", context: ["usedCache": String(screenState == .widget)])
  }

  func authenticationSucceeded() async {
    screenState = .checkingAuthentication

    await initialize()
    if screenState == .widget {
      startPeriodicRefresh()
    }
  }

  func retryStartup() async {
    screenState = .checkingAuthentication
    await start()
  }

  @discardableResult
  func initializeFromCache() -> Bool {
    let today = DateFormatter.yyyyMMdd.string(from: Date())
    do {
      guard let bootstrap = try repository.cachedInitialization(calendarDate: today) else {
        return false
      }
      try validateBootstrap(bootstrap, requestedDate: nil)
      applyBootstrap(bootstrap, resetCurrentCategory: true)
      context.currentDayRecord = DayRecord(calendarDate: today, plan: bootstrap.dayRecords[0].plan)
      currentState = ActiveState()
      screenState = .widget
      lastError = nil
      WidgetLogger.debug(
        "Widget activated from cached bootstrap",
        context: [
          "cachedCalendarDate": bootstrap.dayRecords[0].calendarDate, "calendarDate": today,
        ])
      return true
    } catch {
      WidgetLogger.error(
        "Cached bootstrap is unavailable",
        context: ["calendarDate": today, "error": String(describing: error)])
      return false
    }
  }

  func initialize() async {
    do {
      try await loadData()
      await dispatch(.initialize)
      screenState = .widget
      lastError = nil
    } catch {
      lastError = String(describing: error)
      if isAuthenticationError(error) {
        screenState = .login
      } else {
        screenState = .error(lastError ?? "Unable to load widget data")
      }
      WidgetLogger.error(
        "Initialization failed", context: ["error": lastError ?? "unknown"])
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
    applyBootstrap(bootstrap, resetCurrentCategory: true)
  }

  private func applyBootstrap(_ bootstrap: InitResponse, resetCurrentCategory: Bool) {
    context.categories = bootstrap.categories
    context.settings = bootstrap.settings
    context.currentPlannedBlocks = bootstrap.dayRecords[0].plan
    context.currentDayRecord = bootstrap.dayRecords[0]

    let currentPlannedBlock =
      TimeLogic.getCurrentPlannedBlock(
        at: Date(), from: context.currentPlannedBlocks)
      ?? TimeLogic.getNextPlannedBlock(at: Date(), from: context.currentPlannedBlocks)
    let plannedCategory = context.categories.first { $0.id == currentPlannedBlock?.categoryId }
    context.plannedCategory = plannedCategory

    if resetCurrentCategory || context.currentCategory == nil
      || !context.categories.contains(where: { $0.id == context.currentCategory?.id })
    {
      context.currentCategory = plannedCategory
    }
  }

  private func validateBootstrap(_ bootstrap: InitResponse, requestedDate: String?) throws {
    guard let dayRecord = bootstrap.dayRecords.first else {
      throw StorageError.invalidContract("bootstrap does not contain a day record")
    }
    if let requestedDate, dayRecord.calendarDate != requestedDate {
      throw StorageError.invalidContract("day_records calendar date does not match request")
    }
    let categoryIds = Set(bootstrap.categories.map(\.id))
    for block in dayRecord.plan {
      guard categoryIds.contains(block.categoryId), block.durationMinutes > 0,
        TimeLogic.parseSeconds(from: block.startTime) != nil
      else {
        throw StorageError.invalidContract("invalid planned block \(block.id)")
      }
    }
  }

  func handleSelectCategory(_ category: Category) async {
    await dispatch(.selectCategory(category))
  }
  func handlePrimaryAction() async { await dispatch(.primaryAction) }
  func adjustOffset(minutes: Int) async { await dispatch(.adjustOffset(minutes)) }

  func synchronize() async {
    WidgetLogger.debug("Widget synchronization requested")

    guard repository.getAuthToken() != nil else {
      lastError = String(describing: StorageError.unauthorized)
      screenState = .login
      WidgetLogger.error("Synchronization requires authentication")
      return
    }

    var synchronizationError: Error?
    do {
      try await repository.synchronize()
    } catch {
      synchronizationError = error
    }

    if let synchronizationError {
      lastError = String(describing: synchronizationError)
      WidgetLogger.error("Synchronization failed", context: ["error": lastError!])
      if isAuthenticationError(synchronizationError) {
        screenState = .login
      }
    } else {
      refreshFromCache()
      lastError = nil
    }
  }

  private func isAuthenticationError(_ error: Error) -> Bool {
    if case StorageError.unauthorized = error {
      return true
    }
    if case APIError.unauthorized = error {
      return true
    }
    return String(describing: error).contains("unauthorized")
  }

  func clearLocalData() async {
    WidgetLogger.debug("Clearing local widget data")
    do {
      try repository.clearLocalData()
      context = WidgetContext()
      currentState = InitializingState()
      lastError = nil
    } catch {
      lastError = String(describing: error)
      WidgetLogger.error("Failed to clear local widget data", context: ["error": lastError!])
    }
  }

  func logout() async {
    await clearLocalData()
    do {
      try await repository.clearAuth()
      screenState = .login
    } catch {
      lastError = String(describing: error)
      WidgetLogger.error("Logout failed", context: ["error": lastError!])
    }
  }

  private func refreshFromCache() {
    let today = DateFormatter.yyyyMMdd.string(from: Date())
    do {
      guard let bootstrap = try repository.cachedInitialization(calendarDate: today) else {
        return
      }
      try validateBootstrap(bootstrap, requestedDate: today)
      applyBootstrap(bootstrap, resetCurrentCategory: displayState == .initializing)
      if displayState == .initializing {
        currentState = ActiveState()
      }
    } catch {
      WidgetLogger.error(
        "Cached bootstrap refresh was ignored",
        context: ["calendarDate": today, "error": String(describing: error)])
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
        context.eventLocalTimestamps[eventResult.event.clientEventId] =
          eventResult.event.occurredAtLocal
        context.lastEventClientId = eventResult.clientEventId
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
        context.eventLocalTimestamps[eventResult.event.clientEventId] =
          eventResult.event.occurredAtLocal
        context.lastEventClientId = eventResult.clientEventId
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
          correctedAt: correctedAt,
          targetOccurredAtLocal: context.eventLocalTimestamps[targetClientEventId]
        )
        context.eventLocalTimestamps[eventResult.event.clientEventId] =
          eventResult.event.occurredAtLocal
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
    let message =
      "[EFFECT] \(effectName) eventCategory=\(categoryDescription(eventCategory)); "
      + "state=\(stateDescription(displayState)); "
      + "currentCategory=\(categoryDescription(context.currentCategory)); "
      + "plannedCategory=\(categoryDescription(context.plannedCategory)); "
      + "plannedBlock=\(plannedBlockDescription(plannedBlock)); "
      + "offsetSeconds=\(context.offsetSeconds)"
    WidgetLogger.debug("Effect applied", context: ["details": message])
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

  // MARK: - Heartbeat

  private func setupTicker() {
    ticker = Timer.publish(
      every: TimeInterval(widgetTickerIntervalSeconds), on: .main, in: .common)
      .autoconnect()
      .sink { [weak self] _ in
        guard let self = self else { return }
        self.tick += widgetTickerIntervalSeconds
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

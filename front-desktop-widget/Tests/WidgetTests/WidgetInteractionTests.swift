import Foundation

@MainActor
final class WidgetInteractionTests: WidgetTestCase {
func test_selectCategory_logsTransition() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
       expectedType: "transition",
       expectedIncomingId: Fixtures.categoryA.id,
    )
  }
func test_afterConfirmation_returnsToActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: 1))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.primaryAction)
    try assert(h.store.displayState == .active, "Should return to active after dispatch")
  }
func test_multipleSelectCategories() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))
    try h.assertSubmitEventsCount(2)
  }
func test_adjustOffset_backward() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()
    let beforeAdjust = h.store.lastEventTime

    await h.store.dispatch(.adjustOffset(5))

    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
       expectedType: "amendment",
       expectedIncomingId: nil,
    )
    let expectedTime = beforeAdjust.addingTimeInterval(-5 * 60)
    try assert(abs(h.store.lastEventTime.timeIntervalSince(expectedTime)) < 2, "Offset time should be retroactive")
  }
func test_adjustOffset_negativeIsBlocked() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.adjustOffset(-5))

    try h.assertNoSubmitEvents()
    try assert(h.store.offsetMinutes == 0, "Offset must not become negative")
  }
func test_adjustOffset_multipleAccumulate() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()
    let initialEventTime = h.store.lastEventTime

    await h.store.dispatch(.adjustOffset(5))
    await h.store.dispatch(.adjustOffset(5))

    try h.assertSubmitEventsCount(2)
    try h.assertSubmitEventDetails(index: 0, expectedType: "amendment", expectedIncomingId: nil)
    try h.assertSubmitEventDetails(index: 1, expectedType: "amendment", expectedIncomingId: nil)
    let amendments = h.mock.submitEventsCalls.map { $0.events[0] }
    try assertEqual(amendments[0].targetClientEventId, amendments[1].targetClientEventId)
    try assert(
      amendments[0].correctedAtLocal?.suffix(6) == amendments[1].correctedAtLocal?.suffix(6),
      "Amendments should preserve the target event timezone offset"
    )
    try assert(
      amendments[0].correctedAtLocal != amendments[0].occurredAtLocal,
      "Amendments should carry the corrected wall-clock time"
    )
    try assert(h.store.offsetMinutes == 10, "Offset should accumulate to 10")
  }
func test_adjustOffset_noCurrentCategory_noSubmit() async throws {
     let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initializeAndResetCalls()

    await h.store.dispatch(.adjustOffset(5))

    try h.assertNoSubmitEvents()
  }
func test_adjustOffset_withoutPersistedEvent_doesNotMutateContext() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initializeAndResetCalls()
    let originalTime = h.store.lastEventTime
    await h.store.dispatch(.adjustOffset(5))
    try assertEqual(h.store.offsetMinutes, 0)
    try assertEqual(h.store.lastEventTime, originalTime)
  }
func test_returnToPlan_whenOffSchedule() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))
    h.mock.resetCalls()

    await h.store.dispatch(.returnToPlan)

    if h.store.plannedCategory != nil {
      try h.assertSubmitEventsCount(1)
      try h.assertSubmitEventDetails(
        index: 0,
        expectedType: "transition",
        expectedIncomingId: Fixtures.categoryA.id
      )
    }
  }
func test_returnToPlan_whenOnSchedule_noSubmit() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.returnToPlan)

    try h.assertNoSubmitEvents()
  }
func test_returnToPlan_noPlannedCategory_noSubmit() async throws {
     let h = WidgetTestHarness(existingRecord: Fixtures.record())
    await h.initializeAndResetCalls()

    await h.store.dispatch(.returnToPlan)

    try h.assertNoSubmitEvents()
  }
func test_flow_selectMultipleCategoriesInSequence() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    try h.assertSubmitEventsCount(3)
     try h.assertSubmitEventDetails(index: 0, expectedType: "transition", expectedIncomingId: Fixtures.categoryA.id)
    try h.assertSubmitEventDetails(index: 1, expectedType: "transition", expectedIncomingId: Fixtures.categoryB.id)
    try h.assertSubmitEventDetails(index: 2, expectedType: "transition", expectedIncomingId: Fixtures.categoryA.id)
  }
func test_flow_offsetThenSelectCategory() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()
    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    h.mock.resetCalls()

    await h.store.dispatch(.adjustOffset(5))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    try h.assertSubmitEventsCount(2)
     try h.assertSubmitEventDetails(index: 0, expectedType: "amendment", expectedIncomingId: nil)
    try h.assertSubmitEventDetails(index: 1, expectedType: "transition", expectedIncomingId: Fixtures.categoryB.id)
  }
func test_submitEventsThrows_doesNotCrash() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    h.mock.shouldThrowOnSubmitEvents = true

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    try h.assertSubmitEventsCount(1)
    // Store should handle error gracefully
  }
func test_submitEventsThrows_rollsBackOptimisticTransition() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    let originalCategory = h.store.currentCategory
    h.mock.shouldThrowOnSubmitEvents = true

    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    try assertEqual(h.store.currentCategory?.id, originalCategory?.id)
    try assert(h.store.lastError != nil, "Submission failure must be exposed")
  }
func test_selectCategory_inActive_staysActive() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock())
    await h.initializeAndResetCalls()
    try assert(h.store.displayState == .active, "Should start in active")

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    try assert(h.store.displayState == .active, "Should remain active after selection")
  }
func test_submitEvents_useCorrectCalendarDate() async throws {
    let record = Fixtures.record()
    let h = WidgetTestHarness(existingRecord: record)
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))

    let calls = h.mock.submitEventsCalls
    try assert(calls.count > 0, "Should have submitEvents calls")
    try assertEqual(calls.first?.calendarDate, record.calendarDate)
  }
func test_transitionEvent_populatesEventFields() async throws {
    // Verify that transition events contain category IDs (both incoming and outgoing are the same)
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    try h.assertSubmitEventsCount(1)
    let calls = h.mock.submitEventsCalls
    let event = calls[0].events[0]
    try assertEqual(event.eventType, "transition")
    try assertEqual(event.categoryId, Fixtures.categoryB.id)
  }
func test_confirmationEvent_populatesEventFields() async throws {
    let h = WidgetTestHarness(existingRecord: Fixtures.recordWithCurrentBlock(categoryId: Fixtures.categoryA.id))
    await h.initializeAndResetCalls()

    await h.store.dispatch(.primaryAction)

    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
      expectedType: "confirmation",
      expectedIncomingId: Fixtures.categoryA.id
    )
    try assert(h.store.displayState == .active, "Should remain active when no boundary")
  }
func test_pomodoroCompleted_sendsConfirmationEvent() async throws {
    let pomodoroCategory = Category(
      id: 10,
      name: "Pomodoro Task",
      color: "#000000",
      pomodoroConfig: PomodoroConfig(workDuration: 60, restDuration: 60),
      createdAt: Fixtures.now,
      updatedAt: Fixtures.now
    )
    let plannedBlock = PlannedBlock(
      categoryId: pomodoroCategory.id, startTime: "00:00:00", durationMinutes: 24 * 60)
    let dayRecord = DayRecord(
      calendarDate: Fixtures.today,
      plan: [plannedBlock]
    )
    let h = WidgetTestHarness(categories: [pomodoroCategory], existingRecord: dayRecord)
    await h.initializeAndResetCalls()
    h.store.context.pomodoroPhase = .work
    h.store.context.pomodoroElapsed = 59

    let result = h.store.currentState.onTick(
      context: h.store.context,
      currentPlannedBlock: plannedBlock
    )
    await h.store.apply(result)

    try h.assertSubmitEventsCount(1)
    try h.assertSubmitEventDetails(
      index: 0,
      expectedType: "confirmation",
      expectedIncomingId: pomodoroCategory.id
    )
  }
@MainActor
  func test_confirmationAtCategoryBoundary_logsTransitionWhenCategoryChanges() async throws {
    var context = WidgetContext()
    context.currentCategory = Fixtures.categoryA
    context.plannedCategory = Fixtures.categoryB

    let result = confirmationResult(context: context, nextState: ActiveState())

    let transitionCategoryIds = result.effects.compactMap { effect -> Int? in
      guard case .logTransition(let category, _) = effect else { return nil }
      return category.id
    }
    try assertEqual(transitionCategoryIds, [Fixtures.categoryB.id])
  }
@MainActor
  func test_confirmationWithoutCategoryChange_logsTransition() async throws {
    var context = WidgetContext()
    context.currentCategory = Fixtures.categoryA
    context.plannedCategory = Fixtures.categoryA

    let result = confirmationResult(context: context, nextState: ActiveState())

    let transitionCategoryIds = result.effects.compactMap { effect -> Int? in
      guard case .logTransition(let category, _) = effect else { return nil }
      return category.id
    }
    try assertEqual(transitionCategoryIds, [Fixtures.categoryA.id])
  }
func test_submitEventsCall_usesCorrectCalendarDateAndEventSequence() async throws {
    let record = Fixtures.record()
    let h = WidgetTestHarness(existingRecord: record)
    await h.initializeAndResetCalls()

    await h.store.dispatch(.selectCategory(Fixtures.categoryA))
    await h.store.dispatch(.selectCategory(Fixtures.categoryB))

    let calls = h.mock.submitEventsCalls
    try assert(calls.count == 2, "Should have 2 submitEvents calls")

    // Verify first call
    let (date1, events1) = calls[0]
    try assertEqual(date1, record.calendarDate)
    try assertEqual(events1.count, 1)
    try assertEqual(events1[0].categoryId, Fixtures.categoryA.id)
    try assert(events1[0].occurredAtLocal.count == 25, "Local timestamp should include numeric offset")
    try assert(events1[0].occurredAtLocal.last == "0", "Local timestamp should be RFC3339")

    // Verify second call
    let (date2, events2) = calls[1]
    try assertEqual(date2, record.calendarDate)
    try assertEqual(events2.count, 1)
    try assertEqual(events2[0].categoryId, Fixtures.categoryB.id)
  }
  static func testMethods() -> [TestCase] {
    let tests = WidgetInteractionTests()
    return [
      ("test_selectCategory_logsTransition", { try await tests.test_selectCategory_logsTransition() }),
      ("test_afterConfirmation_returnsToActive", { try await tests.test_afterConfirmation_returnsToActive() }),
      ("test_multipleSelectCategories", { try await tests.test_multipleSelectCategories() }),
      ("test_adjustOffset_backward", { try await tests.test_adjustOffset_backward() }),
      ("test_adjustOffset_negativeIsBlocked", { try await tests.test_adjustOffset_negativeIsBlocked() }),
      ("test_adjustOffset_multipleAccumulate", { try await tests.test_adjustOffset_multipleAccumulate() }),
      ("test_adjustOffset_noCurrentCategory_noSubmit", { try await tests.test_adjustOffset_noCurrentCategory_noSubmit() }),
      ("test_adjustOffset_withoutPersistedEvent_doesNotMutateContext", { try await tests.test_adjustOffset_withoutPersistedEvent_doesNotMutateContext() }),
      ("test_returnToPlan_whenOffSchedule", { try await tests.test_returnToPlan_whenOffSchedule() }),
      ("test_returnToPlan_whenOnSchedule_noSubmit", { try await tests.test_returnToPlan_whenOnSchedule_noSubmit() }),
      ("test_returnToPlan_noPlannedCategory_noSubmit", { try await tests.test_returnToPlan_noPlannedCategory_noSubmit() }),
      ("test_flow_selectMultipleCategoriesInSequence", { try await tests.test_flow_selectMultipleCategoriesInSequence() }),
      ("test_flow_offsetThenSelectCategory", { try await tests.test_flow_offsetThenSelectCategory() }),
      ("test_submitEventsThrows_doesNotCrash", { try await tests.test_submitEventsThrows_doesNotCrash() }),
      ("test_submitEventsThrows_rollsBackOptimisticTransition", { try await tests.test_submitEventsThrows_rollsBackOptimisticTransition() }),
      ("test_selectCategory_inActive_staysActive", { try await tests.test_selectCategory_inActive_staysActive() }),
      ("test_submitEvents_useCorrectCalendarDate", { try await tests.test_submitEvents_useCorrectCalendarDate() }),
      ("test_transitionEvent_populatesEventFields", { try await tests.test_transitionEvent_populatesEventFields() }),
      ("test_confirmationEvent_populatesEventFields", { try await tests.test_confirmationEvent_populatesEventFields() }),
      ("test_pomodoroCompleted_sendsConfirmationEvent", { try await tests.test_pomodoroCompleted_sendsConfirmationEvent() }),
      ("test_confirmationAtCategoryBoundary_logsTransitionWhenCategoryChanges", { try await tests.test_confirmationAtCategoryBoundary_logsTransitionWhenCategoryChanges() }),
      ("test_confirmationWithoutCategoryChange_logsTransition", { try await tests.test_confirmationWithoutCategoryChange_logsTransition() }),
      ("test_submitEventsCall_usesCorrectCalendarDateAndEventSequence", { try await tests.test_submitEventsCall_usesCorrectCalendarDateAndEventSequence() }),
    ]
  }
}

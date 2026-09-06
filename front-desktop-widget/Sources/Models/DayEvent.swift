import Foundation

struct DayEvent: Codable {
  let clientEventId: String
  let eventType: String  // "confirmation" | "transition"
  let categoryId: Int
  let occurredAt: Date
  let targetClientEventId: String?
  let correctedAt: Date?

  enum CodingKeys: String, CodingKey {
    case clientEventId = "client_event_id"
    case eventType = "event_type"
    case categoryId = "category_id"
    case occurredAt = "occurred_at"
    case targetClientEventId = "target_client_event_id"
    case correctedAt = "corrected_at"
  }

  init(
    clientEventId: String = UUID().uuidString,
    eventType: String,
    categoryId: Int,
    occurredAt: Date,
    targetClientEventId: String? = nil,
    correctedAt: Date? = nil
  ) {
    self.clientEventId = clientEventId
    self.eventType = eventType
    self.categoryId = categoryId
    self.occurredAt = occurredAt
    self.targetClientEventId = targetClientEventId
    self.correctedAt = correctedAt
  }
}

struct DayEventsRequest: Codable {
  let deviceId: Int
  let events: [DayEvent]

  enum CodingKeys: String, CodingKey {
    case deviceId = "device_id"
    case events
  }

  init(deviceId: Int = 0, events: [DayEvent]) {
    self.deviceId = deviceId
    self.events = events
  }
}

struct DayEventsResponse: Codable {
  let acceptedEvents: [AcceptedEvent]
  let duplicateClientEventIds: [String]
  let actualBlocks: [ActualBlock]
  let createdEvents: [CreatedEvent]

  enum CodingKeys: String, CodingKey {
    case acceptedEvents = "accepted_events"
    case duplicateClientEventIds = "duplicate_client_event_ids"
    case createdEvents = "created_events"
    case actualBlocks = "actual_blocks"
  }

  init(
    acceptedEvents: [AcceptedEvent] = [],
    duplicateClientEventIds: [String] = [],
    createdEvents: [CreatedEvent] = [],
    actualBlocks: [ActualBlock]
  ) {
    self.acceptedEvents = acceptedEvents
    self.duplicateClientEventIds = duplicateClientEventIds
    self.createdEvents = createdEvents
    self.actualBlocks = actualBlocks
  }
}

struct AcceptedEvent: Codable {
  let clientEventId: String
  let eventType: String
  let categoryId: Int
  let occurredAt: Date
  enum CodingKeys: String, CodingKey {
    case clientEventId = "client_event_id"
    case eventType = "event_type"
    case categoryId = "category_id"
    case occurredAt = "occurred_at"
  }
}

struct CreatedEvent: Codable {
  let id: Int
  let eventType: String
  let categoryId: Int
  let occurredAt: Date

  enum CodingKeys: String, CodingKey {
    case id
    case eventType = "event_type"
    case categoryId = "category_id"
    case occurredAt = "occurred_at"
  }
}

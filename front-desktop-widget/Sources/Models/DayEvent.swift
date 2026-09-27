import Foundation

struct DayEvent: Codable {
  let clientEventId: String
  let eventType: String  // "confirmation" | "transition"
  let categoryId: Int?
  let occurredAt: Date
  let occurredAtLocal: String
  let targetClientEventId: String?
  let correctedAt: Date?
  let correctedAtLocal: String?

  enum CodingKeys: String, CodingKey {
    case clientEventId = "client_event_id"
    case eventType = "event_type"
    case categoryId = "category_id"
    case occurredAt = "occurred_at"
    case occurredAtLocal = "occurred_at_local"
    case targetClientEventId = "target_client_event_id"
    case correctedAt = "corrected_at"
    case correctedAtLocal = "corrected_at_local"
  }

  init(
    clientEventId: String = UUID().uuidString,
    eventType: String,
    categoryId: Int?,
    occurredAt: Date,
    occurredAtLocal: String,
    targetClientEventId: String? = nil,
    correctedAt: Date? = nil,
    correctedAtLocal: String? = nil
  ) {
    self.clientEventId = clientEventId
    self.eventType = eventType
    self.categoryId = categoryId
    self.occurredAt = occurredAt
    self.occurredAtLocal = occurredAtLocal
    self.targetClientEventId = targetClientEventId
    self.correctedAt = correctedAt
    self.correctedAtLocal = correctedAtLocal
  }
}

struct DayEventsRequest: Codable {
  let deviceId: Int
  let events: [DayEvent]

  enum CodingKeys: String, CodingKey {
    case deviceId = "device_id"
    case events
  }

  init(deviceId: Int, events: [DayEvent]) {
    self.deviceId = deviceId
    self.events = events
  }
}

struct DayEventsResponse: Decodable {
  let acceptedEvents: [AcceptedEvent]
  let duplicateClientEventIds: [String]
  let calendarDate: String

  enum CodingKeys: String, CodingKey {
    case acceptedEvents = "accepted_events"
    case duplicateClientEventIds = "duplicate_client_event_ids"
    case calendarDate = "calendar_date"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    acceptedEvents = try container.decode([AcceptedEvent].self, forKey: .acceptedEvents)
    duplicateClientEventIds = try container.decode([String].self, forKey: .duplicateClientEventIds)
    calendarDate = try container.decode(String.self, forKey: .calendarDate)
  }

  init(
    acceptedEvents: [AcceptedEvent] = [],
    duplicateClientEventIds: [String] = [],
    calendarDate: String = ""
  ) {
    self.acceptedEvents = acceptedEvents
    self.duplicateClientEventIds = duplicateClientEventIds
    self.calendarDate = calendarDate
  }
}

struct AcceptedEvent: Codable {
  let clientEventId: String
  let eventType: String
  let categoryId: Int?
  let occurredAt: Date
  let occurredAtLocal: String?
  enum CodingKeys: String, CodingKey {
    case clientEventId = "client_event_id"
    case eventType = "event_type"
    case categoryId = "category_id"
    case occurredAt = "occurred_at"
    case occurredAtLocal = "occurred_at_local"
  }

}

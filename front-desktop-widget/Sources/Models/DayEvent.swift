import Foundation

struct DayEvent: Codable {
  let clientEventId: String
  let eventType: String  // "confirmation" | "transition" | "note"
  let categoryId: Int?
  let text: String?
  let occurredAt: Date
  let targetClientEventId: String?
  let correctedAt: Date?

  enum CodingKeys: String, CodingKey {
    case clientEventId = "client_event_id"
    case eventType = "event_type"
    case categoryId = "category_id"
    case text
    case occurredAt = "occurred_at"
    case targetClientEventId = "target_client_event_id"
    case correctedAt = "corrected_at"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    clientEventId = try container.decode(String.self, forKey: .clientEventId)
    eventType = try container.decode(String.self, forKey: .eventType)
    categoryId = try container.decodeIfPresent(Int.self, forKey: .categoryId)
    text = try container.decodeIfPresent(String.self, forKey: .text)
    occurredAt = try container.decode(Date.self, forKey: .occurredAt)
    targetClientEventId = try container.decodeIfPresent(String.self, forKey: .targetClientEventId)
    correctedAt = try container.decodeIfPresent(Date.self, forKey: .correctedAt)

  }

  init(
    clientEventId: String = UUID().uuidString,
    eventType: String,
    categoryId: Int?,
    text: String? = nil,
    occurredAt: Date,
    targetClientEventId: String? = nil,
    correctedAt: Date? = nil
  ) {
    self.clientEventId = clientEventId
    self.eventType = eventType
    self.categoryId = categoryId
    self.text = text
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
  let text: String? = nil
  let occurredAt: Date
  enum CodingKeys: String, CodingKey {
    case clientEventId = "client_event_id"
    case eventType = "event_type"
    case categoryId = "category_id"
    case text
    case occurredAt = "occurred_at"
  }

}

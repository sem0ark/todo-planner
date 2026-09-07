import Foundation

struct DayEvent: Codable {
  let clientEventId: String
  let eventType: String  // "confirmation" | "transition"
  let categoryId: Int?
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
    categoryId: Int?,
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

  init(deviceId: Int, events: [DayEvent]) {
    self.deviceId = deviceId
    self.events = events
  }
}

struct DayEventsResponse: Decodable {
  let acceptedEvents: [AcceptedEvent]
  let duplicateClientEventIds: [String]
  let calendarDate: String
  let dayTemplateId: Int?
  let plan: [PlannedBlock]
  let actual: [ActualBlock]
  let createdAt: Date
  let updatedAt: Date

  enum CodingKeys: String, CodingKey {
    case acceptedEvents = "accepted_events"
    case duplicateClientEventIds = "duplicate_client_event_ids"
    case calendarDate = "calendar_date"
    case dayTemplateId = "day_template_id"
    case plan, actual
    case createdAt = "created_at"
    case updatedAt = "updated_at"
  }

  init(from decoder: Decoder) throws {
    let container = try decoder.container(keyedBy: CodingKeys.self)
    acceptedEvents = try container.decode([AcceptedEvent].self, forKey: .acceptedEvents)
    duplicateClientEventIds = try container.decode([String].self, forKey: .duplicateClientEventIds)
    calendarDate = try container.decode(String.self, forKey: .calendarDate)
    dayTemplateId = try container.decodeIfPresent(Int.self, forKey: .dayTemplateId)
    plan = try container.decode([PlannedBlock].self, forKey: .plan)
    actual = try container.decode([ActualBlock].self, forKey: .actual)
    createdAt = try container.decode(Date.self, forKey: .createdAt)
    updatedAt = try container.decode(Date.self, forKey: .updatedAt)
  }

  init(
    acceptedEvents: [AcceptedEvent] = [],
    duplicateClientEventIds: [String] = [],
    calendarDate: String = "",
    dayTemplateId: Int? = nil,
    plan: [PlannedBlock] = [], actual: [ActualBlock] = [],
    createdAt: Date = Date(), updatedAt: Date = Date()
  ) {
    self.acceptedEvents = acceptedEvents
    self.duplicateClientEventIds = duplicateClientEventIds
    self.calendarDate = calendarDate
    self.dayTemplateId = dayTemplateId
    self.plan = plan
    self.actual = actual
    self.createdAt = createdAt
    self.updatedAt = updatedAt
  }
}

struct AcceptedEvent: Codable {
  let clientEventId: String
  let eventType: String
  let categoryId: Int?
  let occurredAt: Date
  enum CodingKeys: String, CodingKey {
    case clientEventId = "client_event_id"
    case eventType = "event_type"
    case categoryId = "category_id"
    case occurredAt = "occurred_at"
  }
}

import Foundation

struct DayEvent: Codable {
    let eventType: String // "confirmation" | "transition"
    let outgoingCategoryId: Int?
    let incomingCategoryId: Int?
    let occurredAt: Date

    enum CodingKeys: String, CodingKey {
        case eventType = "event_type"
        case outgoingCategoryId = "outgoing_category_id"
        case incomingCategoryId = "incoming_category_id"
        case occurredAt = "occurred_at"
    }
}

struct DayEventsRequest: Codable {
    let events: [DayEvent]
}

struct DayEventsResponse: Codable {
    let createdEvents: [CreatedEvent]
    let actualBlocks: [ActualBlock]

    enum CodingKeys: String, CodingKey {
        case createdEvents = "created_events"
        case actualBlocks = "actual_blocks"
    }
}

struct CreatedEvent: Codable {
    let id: Int
    let eventType: String
    let outgoingCategoryId: Int?
    let incomingCategoryId: Int?
    let occurredAt: Date

    enum CodingKeys: String, CodingKey {
        case id
        case eventType = "event_type"
        case outgoingCategoryId = "outgoing_category_id"
        case incomingCategoryId = "incoming_category_id"
        case occurredAt = "occurred_at"
    }
}
